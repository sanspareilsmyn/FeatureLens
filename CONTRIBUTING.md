# Contributing to FeatureLens

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

To contribute code, you'll need a local development environment. We use Docker Compose to manage external dependencies like Kafka, Zookeeper, Prometheus, Grafana, and AKHQ. The **FeatureLens application itself is run locally** on your host machine for easier development and debugging.

1.  **Prerequisites:**
    *   Go (Version 1.22+ recommended - check `go.mod`)
    *   Git
    *   Docker & Docker Compose (Essential for running infrastructure components)
    *   (Optional but Recommended) Python & pip (for pre-commit)

2.  **Fork & Clone:**
    *   Fork the repository on GitHub.
    *   Clone your fork locally:
        ```bash
        git clone https://github.com/<your-username>/featurelens.git
        cd featurelens
        ```

3.  **Configure Local FeatureLens Application:**
    *   **Crucial Step:** You need to configure the FeatureLens application (which will run locally) to connect to the Kafka instance running inside Docker.
    *   Open the development configuration file: `configs/config.dev.yaml`.
    *   Locate the `kafka:` section.
    *   Ensure the `brokers` list points to **Kafka exposed on the host machine via Docker Compose**:
        ```yaml
        kafka:
          brokers: ["localhost:9092"] # <-- MUST be this for local development against Docker Kafka
          topic: feature-stream       # Or your chosen topic name
          groupID: featurelens-dev-group # Use a development group ID
          # ... other settings
        ```
    *   Verify the `topic` name matches what you will create in the next step.

4.  **Start External Dependencies via Docker Compose:**
    *   Navigate to the repository root directory (where `docker-compose.yml` is located).
    *   Start Kafka, Zookeeper, Prometheus, Grafana, and AKHQ in the background. **Note:** We are *not* starting `featurelens` via Docker Compose.
        ```bash
        # If you encounter issues due to previous runs, clean up volumes first:
        # docker-compose down -v

        # Start required infrastructure services
        docker-compose up -d kafka prometheus grafana akhq zookeeper
        ```
    *   This will start the following services, accessible from your host machine:
        *   Kafka: `localhost:9092`
        *   Prometheus: `http://localhost:9090`
        *   Grafana: `http://localhost:3000` (Initial login: admin/admin)
        *   AKHQ (Kafka UI): `http://localhost:8080`
    *   Wait a minute or two for all services (especially Kafka) to initialize fully. You can check logs if needed: `docker-compose logs -f kafka`
    *   **To stop the Docker environment:**
        ```bash
        docker-compose down
        ```
    *   **(Note:** Docker can consume significant system resources. Ensure you have enough RAM/CPU allocated.)

5.  **Create Initial Kafka Topic (Important!):**
    *   FeatureLens needs its input topic to exist in Kafka. Create it *after* starting the Docker services (Step 4).
    *   **Using AKHQ UI (Recommended):**
        1.  Open `http://localhost:8080`.
        2.  Navigate to "Topics" -> "Create Topic".
        3.  Enter the **topic name** exactly as specified in `configs/config.dev.yaml` (e.g., `feature-stream`).
        4.  Use defaults (1 partition, replication factor 1 for this local setup).
        5.  Click "Create".
    *   **(Alternative) Using Command Line:**
        ```bash
        # Execute the kafka-topics command inside the kafka container
        docker-compose exec kafka kafka-topics --create --topic feature-stream --bootstrap-server kafka:9093 --partitions 1 --replication-factor 1
        ```
        *(Replace `feature-stream` if needed. Note we use `kafka:9093` as the command runs *inside* the docker network)*

6.  **Set up Pre-commit Hooks (Highly Recommended):**
    *   Install `pre-commit` tool: `pip install pre-commit` (or `brew install pre-commit`).
    *   Install hooks for this repo:
        ```bash
        pre-commit install
        ```
    *   This runs formatters/linters automatically before each commit (using `.pre-commit-config.yaml`).

7.  **Dependencies & Local Build:**
    *   Fetch Go dependencies for local tooling and IDE support:
        ```bash
        go mod download
        go mod tidy
        ```
    *   Build the FeatureLens application **locally**:
        ```bash
        # Build the executable in the project root (or adjust path as needed)
        go build -o featurelens ./cmd/featurelens
        ```
    *   (Optional) Build a sample producer locally if you have one:
        ```bash
        # go build -o producer ./cmd/producer
        ```

8.  **Running & Testing the System:**
    *   **1. Ensure Docker Services are Running:** Check that Kafka, Prometheus, Grafana etc. are running (`docker-compose ps`).
    *   **2. Run FeatureLens Locally:** In a terminal, start the application binary you built, pointing it to the development configuration file:
        ```bash
        ./featurelens -config configs/config.dev.yaml
        ```
        *   Observe the terminal output. You should see logs indicating:
            *   Successful configuration loading.
            *   Successful pipeline initialization.
            *   Prometheus metrics server starting on `:8081`.
            *   Kafka consumer starting and attempting to connect to `localhost:9092`. It should connect without errors (after Kafka is ready).
    *   **3. (Optional) Start Sample Producer:** If you have a producer, run it *locally* to send test messages to the Kafka topic (`localhost:9092`, `feature-stream`).
    *   **4. Verify Prometheus Connection:**
        *   Open the Prometheus UI: `http://localhost:9090`.
        *   Navigate to **Status -> Targets**.
        *   Look for the `featurelens` job. Its endpoint should be `http://host.docker.internal:8081/metrics` and the **State** should be **UP**.
        *   *Troubleshooting if DOWN:* Ensure your local `./featurelens` process is running, check its logs for errors, verify the `:8081` metrics server started, confirm `prometheus.yml` has the correct target, and check local firewall settings if necessary.
    *   **5. Verify Grafana Dashboard & Data:**
        *   Open the Grafana UI: `http://localhost:3000`.
        *   Login (default: admin/admin).
        *   Navigate to **Dashboards**. You should see a folder named `FeatureLens` (or similar, based on provisioning) containing your pre-configured dashboard (e.g., `FeatureLens Overview`).
        *   Open the dashboard. After sending some test messages via the producer (Step 3) and waiting for Prometheus to scrape and FeatureLens to process, you should see data appearing in the panels.
        *   If the dashboard isn't automatically provisioned, check the Grafana provisioning volume mounts in `docker-compose.yml` and the YAML/JSON file contents in the `grafana/provisioning` directory. You can also manually add the Prometheus data source (URL: `http://prometheus:9090`).
    *   **6. Query Metrics Directly:** You can also query metrics directly in the Prometheus UI (Graph tab) using names like `featurelens_feature_window_count_total`.
    *   **7. Run Unit Tests:** These run independently of the Docker setup:
        ```bash
        go test ./...
        ```

9.  **Stopping the Environment:**
    *   Stop the local `featurelens` application (usually Ctrl+C in its terminal).
    *   Stop the Docker Compose services:
        ```bash
        docker-compose down
        ```
        *(Use `docker-compose down -v` if you want to remove Kafka/Zookeeper data and start fresh next time)*

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
