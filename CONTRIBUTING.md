# Contributing to FeatureLens 🔭렌즈

First off, thank you for considering contributing to FeatureLens! 🎉 We welcome any help to make this project better. Whether it's reporting a bug, proposing a new feature, improving documentation, or writing code, your contribution is valuable.

This document provides guidelines for contributing to FeatureLens. Please take a moment to review it.

## Code of Conduct

While we don't have a formal Code of Conduct yet, we expect all contributors to interact respectfully and constructively. Please be kind and considerate in all discussions and contributions. Harassment or exclusionary behavior will not be tolerated.

## How Can I Contribute?

There are many ways to contribute:

*   **🐛 Reporting Bugs:** If you find a bug, please open an issue on GitHub. Provide as much detail as possible.
*   **✨ Suggesting Enhancements:** Open an issue to discuss new features or improvements.
*   **📝 Improving Documentation:** PRs for documentation improvements are always welcome!
*   **💻 Writing Code:**
    1.  **Discuss First (for significant changes):** Open an issue before starting major work.
    2.  **Follow the Workflow:** See the "Contribution Workflow" section below.

## Getting Started (Development Setup)

To contribute code, you'll need a local development environment. We use Docker Compose to provide a consistent Kafka environment with a Web UI (AKHQ) for testing.

1.  **Prerequisites:**
    *   Go (Version 1.19+ recommended - check `go.mod`)
    *   Git
    *   Python & pip (for pre-commit)
    *   Docker & Docker Compose (essential for the local Kafka environment)

2.  **Fork & Clone:**
    *   Fork the repository on GitHub.
    *   Clone your fork locally:
        ```bash
        git clone https://github.com/<your-username>/featurelens.git
        cd featurelens
        ```

3.  **Set up Local Kafka Environment:**
    *   Navigate to the repository root directory (where `docker-compose.yml` is located).
    *   Start the Kafka cluster and AKHQ UI in the background:
        ```bash
        docker-compose up -d
        ```
    *   This will start Kafka listening on `localhost:9092` (for FeatureLens/producer) and `kafka:9093` (internally), and the AKHQ UI on `http://localhost:8080`. Wait a few moments for the services to fully initialize.
    *   **To stop the Kafka environment:**
        ```bash
        docker-compose down
        ```
    *   **(Note:** Docker can consume significant system resources. Ensure you have enough RAM/CPU allocated.)

4.  **Create Initial Kafka Topic (Important!):**
    *   The sample producer and FeatureLens expect a specific topic to exist. We need to create it manually for the local environment.
    *   **Using AKHQ UI (Recommended):**
        1.  Open your web browser and navigate to the AKHQ UI: `http://localhost:8080`.
        2.  Go to the "Topics" section.
        3.  Click on the "Create Topic" button (or similar).
        4.  Enter the **topic name** defined in the producer code and example configuration (e.g., `feature-stream` - check `cmd/producer/main.go` and `configs/config.dev.yaml` or `configs/config.example.yaml`).
        5.  You can usually leave the number of partitions and replication factor as the default (e.g., 1 partition, replication factor 1 for this local setup).
        6.  Click "Create".
    *   **(Alternative) Using Command Line:**
        ```bash
        # Execute the kafka-topics command inside the kafka container
        docker-compose exec kafka kafka-topics --create --topic feature-stream --bootstrap-server kafka:9093 --partitions 1 --replication-factor 1
        ```
        *(Replace `feature-stream` with the actual topic name if different. Note we use `kafka:9093` for the bootstrap server as this command runs *inside* the docker network)*

5.  **Set up Pre-commit Hooks (Highly Recommended):**
    *   Install `pre-commit` tool: `pip install pre-commit` (or `brew install pre-commit`).
    *   Install hooks for this repo:
        ```bash
        pre-commit install
        ```
    *   This runs formatters/linters automatically before each commit (using `.pre-commit-config.yaml`).

6.  **Dependencies & Build:**
    *   Fetch Go dependencies:
        ```bash
        go mod download
        go mod tidy
        ```
    *   Build FeatureLens:
        ```bash
        go build ./cmd/featurelens # Assuming main package is here
        ```
    *   Build the sample producer (if applicable):
        ```bash
        go build ./cmd/producer # Assuming producer code is here
        ```

7.  **Running Locally:**
    *   **Start Kafka:** Ensure the Kafka cluster is running (`docker-compose up -d`) and the **topic is created (Step 4)**.
    *   **(Optional) Start Sample Producer:** Run the producer to send sample messages to the created topic:
        ```bash
        ./producer # Assuming built producer executable is in root
        ```
        *(Press Ctrl+C to stop the producer)*
    *   **Run FeatureLens:** Start the FeatureLens application, pointing it to your local Kafka and using a configuration file:
        ```bash
        ./featurelens -config configs/config.dev.yaml # Use the dev config (or another appropriate one)
        ```
        *(Ensure `configs/config.dev.yaml` points to `brokers: ["localhost:9092"]` and monitors the created topic, e.g., `feature-stream`)*

8.  **Testing:**
    *   Run unit tests:
        ```bash
        go test ./...
        ```
    *   *(Future: Add integration tests that might use the Docker Compose environment)*

## Contribution Workflow

1.  **Create a Branch:** `git checkout -b <your-branch-name>` from `main`.
2.  **Make Changes:** Write code and documentation.
3.  **Add Tests:** Ensure changes are covered by unit tests.
4.  **Ensure Tests Pass:** `go test ./...`
5.  **Ensure Pre-commit Checks Pass:** Fix issues reported upon `git commit`.
6.  **Commit Changes:** Use Conventional Commits (`git commit -m "feat: ..." `).
7.  **Push Changes:** `git push origin <your-branch-name>`
8.  **Open a Pull Request (PR):** Provide clear title/description, link issues.

## Pull Request Process

1.  **Review:** Maintainer review.
2.  **CI Checks:** Automated build/test/lint checks must pass.
3.  **Discussion & Iteration:** Address feedback.
4.  **Approval & Merge:** Merge into `main` after approval.

## Questions?

Open an issue on GitHub.

Thank you for contributing to FeatureLens! 🙏
