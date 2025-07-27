package registryimpl

import (
	"context"
	"encoding/json"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-infra/coord/internal/client"
	"github.com/ceyewan/gochat/im-infra/coord/registry"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
)

// EtcdServiceRegistry implements the registry.ServiceRegistry interface using etcd.
type EtcdServiceRegistry struct {
	client *client.EtcdClient
	prefix string
	logger clog.Logger

	// Keep track of active sessions for services registered by this instance.
	sessions   map[string]*concurrency.Session
	sessionsMu sync.Mutex
}

// NewEtcdServiceRegistry creates a new etcd-based service registry.
func NewEtcdServiceRegistry(c *client.EtcdClient, prefix string, logger clog.Logger) *EtcdServiceRegistry {
	if prefix == "" {
		prefix = "/services"
	}
	if logger == nil {
		logger = clog.Module("coordination.registry")
	}
	return &EtcdServiceRegistry{
		client:   c,
		prefix:   prefix,
		logger:   logger,
		sessions: make(map[string]*concurrency.Session),
	}
}

// Register a service with a given TTL. The service will be kept alive until the context is canceled or Unregister is called.
func (r *EtcdServiceRegistry) Register(ctx context.Context, service registry.ServiceInfo, ttl time.Duration) error {
	if err := validateServiceInfo(service); err != nil {
		return err
	}
	if ttl <= 0 {
		return client.NewError(client.ErrCodeValidation, "service TTL must be positive", nil)
	}

	// Use a session to manage the lease and keep-alive.
	session, err := concurrency.NewSession(r.client.Client(), concurrency.WithTTL(int(ttl.Seconds())))
	if err != nil {
		return client.NewError(client.ErrCodeConnection, "failed to create etcd session", err)
	}

	serviceKey := r.buildServiceKey(service.Name, service.ID)
	serviceData, err := json.Marshal(service)
	if err != nil {
		_ = session.Close() // Best-effort cleanup
		return client.NewError(client.ErrCodeValidation, "failed to serialize service info", err)
	}

	// Register the service with the lease from the session.
	_, err = r.client.Put(ctx, serviceKey, string(serviceData), clientv3.WithLease(session.Lease()))
	if err != nil {
		_ = session.Close() // Best-effort cleanup
		return client.NewError(client.ErrCodeConnection, "failed to register service", err)
	}

	r.logger.Info("Service registered successfully",
		clog.String("service_name", service.Name),
		clog.String("service_id", service.ID),
		clog.Int64("lease_id", int64(session.Lease())))

	// Store the session to allow for clean unregistration.
	r.sessionsMu.Lock()
	r.sessions[service.ID] = session
	r.sessionsMu.Unlock()

	// The session's keep-alive is running in the background.
	// We can monitor the session's Done channel to know if it has expired.
	go func() {
		<-session.Done()
		r.sessionsMu.Lock()
		delete(r.sessions, service.ID)
		r.sessionsMu.Unlock()
		r.logger.Warn("Service session expired or closed",
			clog.String("service_name", service.Name),
			clog.String("service_id", service.ID))
	}()

	return nil
}

// Unregister removes a service.
func (r *EtcdServiceRegistry) Unregister(ctx context.Context, serviceID string) error {
	if serviceID == "" {
		return client.NewError(client.ErrCodeValidation, "service ID cannot be empty", nil)
	}

	r.sessionsMu.Lock()
	session, ok := r.sessions[serviceID]
	r.sessionsMu.Unlock()

	// If the session is managed by this instance, closing it is the cleanest way to unregister.
	if ok {
		r.logger.Info("Unregistering service by closing its session", clog.String("service_id", serviceID))
		delete(r.sessions, serviceID)
		if err := session.Close(); err != nil {
			return client.NewError(client.ErrCodeConnection, "failed to close session for unregistration", err)
		}
		return nil
	}

	// If the service was registered by another instance, we fall back to deleting the key directly.
	r.logger.Warn("Unregistering service by deleting key (session not found locally)", clog.String("service_id", serviceID))
	key, err := r.findServiceKey(ctx, serviceID)
	if err != nil {
		return err
	}
	if key == "" {
		return client.NewError(client.ErrCodeNotFound, "service not found", nil)
	}

	_, err = r.client.Delete(ctx, key)
	if err != nil {
		return client.NewError(client.ErrCodeConnection, "failed to delete service key", err)
	}

	return nil
}

// Discover finds all instances of a service.
func (r *EtcdServiceRegistry) Discover(ctx context.Context, serviceName string) ([]registry.ServiceInfo, error) {
	if serviceName == "" {
		return nil, client.NewError(client.ErrCodeValidation, "service name cannot be empty", nil)
	}

	prefix := r.buildServicePrefix(serviceName)
	resp, err := r.client.Get(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		return nil, client.NewError(client.ErrCodeConnection, "failed to discover services", err)
	}

	services := make([]registry.ServiceInfo, 0, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		var service registry.ServiceInfo
		if err := json.Unmarshal(kv.Value, &service); err != nil {
			r.logger.Warn("Failed to unmarshal service info, skipping",
				clog.String("key", string(kv.Key)),
				clog.Err(err))
			continue
		}
		services = append(services, service)
	}

	return services, nil
}

// Watch for changes in a service.
func (r *EtcdServiceRegistry) Watch(ctx context.Context, serviceName string) (<-chan registry.ServiceEvent, error) {
	if serviceName == "" {
		return nil, client.NewError(client.ErrCodeValidation, "service name cannot be empty", nil)
	}

	prefix := r.buildServicePrefix(serviceName)
	etcdWatchCh := r.client.Watch(ctx, prefix, clientv3.WithPrefix())
	eventCh := make(chan registry.ServiceEvent, 10)

	go func() {
		defer close(eventCh)
		for resp := range etcdWatchCh {
			if err := resp.Err(); err != nil {
				r.logger.Error("Watch error occurred", clog.String("service_name", serviceName), clog.Err(err))
				// Optionally, send an error event on the channel.
				return
			}
			for _, event := range resp.Events {
				serviceEvent := r.convertEvent(event)
				if serviceEvent != nil {
					select {
					case eventCh <- *serviceEvent:
					case <-ctx.Done():
						return
					}
				}
			}
		}
	}()

	return eventCh, nil
}

// Helper functions
func (r *EtcdServiceRegistry) buildServiceKey(serviceName, serviceID string) string {
	return path.Join(r.prefix, serviceName, serviceID)
}

func (r *EtcdServiceRegistry) buildServicePrefix(serviceName string) string {
	return path.Join(r.prefix, serviceName) + "/"
}

func (r *EtcdServiceRegistry) findServiceKey(ctx context.Context, serviceID string) (string, error) {
	resp, err := r.client.Get(ctx, r.prefix+"/", clientv3.WithPrefix())
	if err != nil {
		return "", client.NewError(client.ErrCodeConnection, "failed to search for service key", err)
	}
	for _, kv := range resp.Kvs {
		if strings.HasSuffix(string(kv.Key), "/"+serviceID) {
			return string(kv.Key), nil
		}
	}
	return "", nil
}

func (r *EtcdServiceRegistry) convertEvent(event *clientv3.Event) *registry.ServiceEvent {
	var service registry.ServiceInfo
	var eventType registry.EventType

	switch event.Type {
	case clientv3.EventTypePut:
		eventType = registry.EventTypePut
		if err := json.Unmarshal(event.Kv.Value, &service); err != nil {
			r.logger.Warn("Failed to unmarshal service info in event", clog.String("key", string(event.Kv.Key)), clog.Err(err))
			return nil
		}
	case clientv3.EventTypeDelete:
		eventType = registry.EventTypeDelete
		// For delete, we can't get the full service info, but we can parse the ID and Name from the key.
		parts := strings.Split(strings.TrimPrefix(string(event.Kv.Key), r.prefix+"/"), "/")
		if len(parts) >= 2 {
			service.Name = parts[0]
			service.ID = parts[1]
		}
	default:
		return nil
	}

	return &registry.ServiceEvent{
		Type:    eventType,
		Service: service,
	}
}

func validateServiceInfo(service registry.ServiceInfo) error {
	if service.ID == "" {
		return client.NewError(client.ErrCodeValidation, "service ID cannot be empty", nil)
	}
	if service.Name == "" {
		return client.NewError(client.ErrCodeValidation, "service name cannot be empty", nil)
	}
	if service.Address == "" {
		return client.NewError(client.ErrCodeValidation, "service address cannot be empty", nil)
	}
	if service.Port <= 0 || service.Port > 65535 {
		return client.NewError(client.ErrCodeValidation, "service port must be between 1 and 65535", nil)
	}
	return nil
}
