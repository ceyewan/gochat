# GoChat Development Workflow

This document outlines the development workflow for the GoChat project. All contributors are expected to follow these guidelines to ensure code quality, consistency, and a smooth development process.

## 1. Branching Strategy

We use a simplified Gitflow model.

-   **`main`**: This branch represents the production-ready code. Direct pushes to `main` are forbidden. Changes are merged into `main` only through pull requests from the `develop` branch.
-   **`develop`**: This is the main development branch. All feature branches are created from `develop` and merged back into it.
-   **Feature Branches**: For any new feature or bugfix, create a new branch from `develop`.
    -   Branch naming convention: `feature/<short-description>` or `fix/<short-description>`.
    -   Example: `feature/add-friend-request` or `fix/login-bug`.

## 2. Development Process

1.  **Create an Issue**: Before starting work, create an issue in the project's issue tracker to describe the feature or bug.
2.  **Create a Branch**: Create a new feature or fix branch from the `develop` branch.
3.  **Develop**: Write your code, following the [Code Style and Conventions](./02_style_guide.md).
4.  **Test**: Write unit and integration tests for your changes. Ensure all tests pass.
5.  **Update Documentation**: If your changes affect any APIs, data models, or architecture, update the relevant documentation in the `docs/` directory.
6.  **Create a Pull Request (PR)**: Once your work is complete and tested, create a pull request to merge your branch into `develop`.
    -   The PR description should clearly explain the changes and reference the issue it resolves.
7.  **Code Review**: At least one other team member must review and approve the PR.
8.  **Merge**: Once approved, the PR can be merged into `develop`.

## 3. Pull Request (PR) Requirements

-   **Clear Title and Description**: The title should be concise, and the description should explain the "what" and "why" of the changes.
-   **Link to Issue**: The PR must be linked to the corresponding issue.
-   **Passing CI Checks**: All automated checks (build, lint, test) must pass.
-   **Code Review Approval**: Must be approved by at least one other developer.

## 4. Versioning

We use Semantic Versioning (SemVer). Version numbers are updated in the `develop` branch as part of the release process.

## 5. Documentation

-   **Documentation-Driven Development**: For new features, it is encouraged to write or update the documentation *before* or *during* development, not after.
-   **Updating Docs**: Any change that impacts the system's behavior, API, or architecture must be reflected in the documentation. This is a mandatory part of the development process.
