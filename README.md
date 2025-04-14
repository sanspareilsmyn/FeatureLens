# FeatureLens

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8.svg)](https://golang.org/) <!-- Updated Go Version -->
[![Build Status](https://img.shields.io/badge/Build-Passing-brightgreen.svg)]() <!-- TODO: Replace with actual CI badge -->
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

**FeatureLens** is a lightweight, high-performance monitoring tool designed specifically for real-time Machine Learning feature pipelines. Built with Go, it focuses on detecting data quality issues and drift efficiently, helping you maintain the health and reliability of your MLOps infrastructure. It calculates key statistics on your live feature data streams and exposes them for monitoring and alerting.

## 🤔 Why FeatureLens?

Machine Learning models are fundamentally dependent on the quality and consistency of their input features. In production environments, real-time feature pipelines are critical but vulnerable to silent failures:

*   **Data Quality Degradation:** Unexpected null values, outliers, or data type changes can creep in due to upstream issues.
*   **Data Drift:** The statistical distribution of live features can deviate significantly from the distributions seen during training, silently harming model performance (Training-Serving Skew).
*   **Pipeline Latency:** Delays in feature calculation or delivery can impact downstream applications.

While various monitoring solutions exist, they can sometimes be resource-intensive, complex, or batch-oriented.

**FeatureLens aims to address these challenges by leveraging Go's strengths:**

*   **🚀 High Performance & Concurrency:** Efficiently monitor multiple Kafka topics/partitions and calculate statistics concurrently with minimal overhead.
*   **📉 Low Resource Footprint:** Consumes significantly fewer resources compared to many Python-based alternatives.
*   **📦 Simple Deployment (Application):** The core FeatureLens logic is built into a single static binary, simplifying local development and potential future containerization.
*   **⏱️ Real-time Focus:** Designed for low-latency processing of streaming data.
*   **📊 Standard Observability:** Integrates seamlessly with standard observability tools like **Prometheus and Grafana** via a `/metrics` endpoint, allowing for flexible visualization and alerting.

By providing timely insights into feature health, FeatureLens helps MLOps teams **detect problems early**, **reduce debugging time**, and **increase trust** in their production ML systems.

## ✨ Features

*   **Kafka Consumer:** Consume feature data messages from specified Apache Kafka topic(s) (assumes JSON format).
*   **Real-time Statistics Calculation (Per Feature):**
    *   Process messages within configurable time windows (e.g., 1-minute tumbling windows).
    *   Calculate basic data quality metrics for specified feature fields:
        *   **Null / Missing Value Rate:** Percentage of null or missing values.
        *   **Mean (Numerical Features):** Average value within the window.
        *   **Variance / Standard Deviation (Numerical Features):** Measure of data dispersion.
        *   **Count:** Total number of messages processed in the window.
*   **Threshold-Based Logging:**
    *   Define acceptable thresholds for calculated metrics in a configuration file.
    *   Log alerts to standard output (stdout) when metrics violate these thresholds.
*   **Metrics Export (Prometheus):**
    *   Expose calculated statistics (Count, Null Rate, Mean, StdDev) and threshold violations as Prometheus metrics on a `/metrics` HTTP endpoint (default port `:8081`).
*   **Configuration:** Load settings (Kafka brokers, topics, features to monitor, window size, thresholds) from a configuration file (e.g., YAML).
*   **Dockerized Infrastructure:** Provides a `docker-compose.yml` to easily run Kafka, Zookeeper, Prometheus, Grafana, and AKHQ for local development and testing.

## 🏗️ Architecture (Local Development)

FeatureLens runs as a **local Go process** during development, connecting to infrastructure components managed by **Docker Compose**. Prometheus (in Docker) scrapes metrics from the local FeatureLens process.
The architecture is designed to facilitate local development and testing, while also being suitable for production deployments.

```text
+-----------------------------------------------------+         +--------------------------------------+
|                 Host Machine (Your Local PC)        |         | Docker Network (e.g., featurelens-net) |
|                                                     |         |                                      |
|  +---------------------+     +--------------------+ |         |  +---------------------------------+ |
|  | FeatureLens App     | --> | localhost:9092     | |         |  | Kafka Container                 | |
|  | (Go Process)        |     | (Kafka External Port)| |         |  | Service: kafka                  | |
|  | ./featurelens ...   |     +--------------------+ |         |  | Port: 9093 (Internal)           | |
|  |                     |                            |         |  +---------------------------------+ |
|  |  - Kafka Consumer   |                            |         |                  ^                   |
|  |  - Parser           |                            |         |                  |                   |
|  |  - Stats Calculator |                            |         |  +---------------------------------+ |
|  |  - Alerter (logs +  |                            |         |  | AKHQ Container                  | |
|  |    Prometheus update)|                            |         |  | Connects to kafka:9093          | |
|  |  - Metrics Server   |                            |         |  +---------------------------------+ |
|  |    (:8081 /metrics) |<------------------------+  |         |                                      |
|  +---------------------+     (Scrapes metrics via|  |         |  +---------------------------------+ |
|                              host.docker.internal:8081) |         |  | Prometheus Container            | |
|                                                     | <-------+  | Service: prometheus             | |
|                                                     |         |  | Port: 9090                      | |
|                                                     |         |  +--------------------^------------+ |
|                                                     |         |                       |              |
|                                                     |         | +---------------------+------------+ |
|                                                     |         | | Grafana Container                | |
|                                                     |         | | Service: grafana                 | |
|                                                     |         | | Port: 3000                       | |
|                                                     |         | | Datasource: http://prometheus:9090 | |
|                                                     |         | +----------------------------------+ |
+-----------------------------------------------------+         +--------------------------------------+
```

**Key Components & Flow:**

1.  **Infrastructure (Docker Compose):** Kafka, Zookeeper, Prometheus, Grafana, and AKHQ run as containers within a dedicated Docker network (`featurelens-net`).
2.  **Kafka:** Listens internally on `kafka:9093` and exposes port `9092` to the host machine for the local FeatureLens application to connect.
3.  **FeatureLens Application (Local Host):**
    *   Runs directly on your machine using `go run` or a compiled binary (`./featurelens`).
    *   **Connects to Kafka** via the exposed port `localhost:9092` (using the configuration specified in `configs/config.dev.yaml`).
    *   Consumes messages from the configured Kafka topic.
    *   Parses incoming JSON messages.
    *   Calculates statistics (Null Rate, Mean, StdDev, Count) within configured time windows.
    *   The **Alerter** component:
        *   Checks calculated statistics against thresholds defined in the configuration.
        *   Logs threshold violations to standard output (stdout).
        *   **Updates internal Prometheus metrics** (Gauges for current stats, Counters for violations) based on the calculation results.
    *   A built-in **HTTP server** (started in `main.go`) listens on port `:8081` and exposes these internal Prometheus metrics via the `/metrics` endpoint.
4.  **Prometheus (Docker):**
    *   Runs inside the Docker network.
    *   Its configuration file (`prometheus.yml`, mounted from the host) defines scrape targets.
    *   For the `featurelens` job, it's configured to scrape `host.docker.internal:8081/metrics`.
    *   Prometheus uses the `host.docker.internal` DNS name to resolve the IP address of the host machine where the FeatureLens app is running.
    *   It periodically fetches (scrapes) the metrics exposed by the FeatureLens app's `/metrics` endpoint.
5.  **Grafana (Docker):**
    *   Runs inside the Docker network.
    *   Configured (typically via provisioning files mounted from the host) to use Prometheus as its data source. The data source URL is `http://prometheus:9090` (using the Prometheus service name within the Docker network).
    *   Loads pre-defined dashboards (also typically provisioned via mounted JSON files) that contain PromQL queries to visualize the metrics collected by Prometheus from FeatureLens.
---
## 🚀 Getting Started (Local Development)

This setup runs infrastructure (Kafka, Prometheus, Grafana) in Docker, while you run and debug the FeatureLens application locally. This is the recommended way for active development.

1.  **Prerequisites:**
    *   Go (Version 1.22+ recommended - check `go.mod`)
    *   Git
    *   Docker & Docker Compose (Essential for running infrastructure components)
    *   (Optional but Recommended) Python & pip (for pre-commit)

2.  **Fork & Clone:**
    *   Fork the repository on GitHub.
    *   Clone your fork locally:
        ```bash
        git clone https://github.com/<your-username>/featurelens.git # Replace with actual repo path
        cd featurelens
        ```

3.  **Configure Local App:**
    *   Ensure the development configuration file exists, typically at `configs/config.dev.yaml`. If not, copy from an example: `cp configs/config.example.yaml configs/config.dev.yaml`.
    *   **Crucially**, open `configs/config.dev.yaml` and verify the Kafka broker settings:
        ```yaml
        kafka:
          brokers: ["localhost:9092"] # MUST point to localhost:9092 for local app connecting to Docker Kafka
          topic: feature-stream       # Ensure this matches the topic you'll create
          groupID: featurelens-dev-group # Use a development-specific group ID
          # ... other relevant Kafka settings ...
        ```
    *   Review other settings like `features` to monitor and their `thresholds`.

4.  **Start Infrastructure via Docker Compose:**
    *   From the project root directory, start the necessary background services:
        ```bash
        # Optional: If you faced issues before, clean up existing containers and volumes
        # docker-compose down -v

        # Start Kafka, Prometheus, Grafana, and supporting services
        docker-compose up -d kafka prometheus grafana akhq zookeeper
        ```
    *   Wait about a minute for all services, especially Kafka, to initialize fully. You can monitor Kafka's readiness with:
        ```bash
        docker-compose logs -f kafka
        ```
        *(Look for messages indicating the broker has started successfully.)*

5.  **Create Initial Kafka Topic:**
    *   The FeatureLens application needs the topic specified in its configuration to exist in Kafka.
    *   **Using AKHQ UI (Recommended):**
        1.  Open `http://localhost:8080` in your browser.
        2.  Navigate to "Topics" -> "Create Topic".
        3.  Enter the exact **topic name** from `configs/config.dev.yaml` (e.g., `feature-stream`).
        4.  Use default settings (1 partition, replication factor 1).
        5.  Click "Create".
    *   **(Alternative) Using Command Line:**
        ```bash
        # Execute inside the running kafka container
        docker-compose exec kafka kafka-topics --create --topic feature-stream --bootstrap-server kafka:9093 --partitions 1 --replication-factor 1
        ```

6.  **(Optional) Set up Pre-commit Hooks:**
    *   If you plan to commit code, it's highly recommended to set up pre-commit hooks for formatting and linting:
        ```bash
        # Install pre-commit tool (if not already installed)
        # pip install pre-commit
        # or brew install pre-commit

        # Install hooks defined in .pre-commit-config.yaml
        pre-commit install
        ```

7.  **Build & Run FeatureLens Locally:**
    *   Ensure Go dependencies are downloaded:
        ```bash
        go mod tidy
        ```
    *   Build the application binary:
        ```bash
        go build -o featurelens ./cmd/featurelens
        ```
    *   Run the compiled application in your terminal:
        ```bash
        ./featurelens -config configs/config.dev.yaml
        ```
        *   Watch the logs for:
            *   Successful initialization messages.
            *   "Starting Prometheus metrics server" log on port `:8081`.
            *   Successful connection logs from the Kafka consumer to `localhost:9092`.
            *   Pipeline components starting their loops.

8.  **(Optional) Send Test Data:**
    *   To see FeatureLens process data and generate metrics, you need to send messages to the Kafka topic (`feature-stream` on `localhost:9092`).
    *   Use a sample producer (`./producer` if available), `kafkacat`, or the "Produce" feature in the AKHQ UI (`http://localhost:8080`). Ensure messages are in the expected JSON format. Example using `kafkacat`:
        ```bash
        echo '{"timestamp": "2023-10-27T10:00:00Z", "user_id": "xyz", "value": 99.9, "feature_x": false}' | kafkacat -P -b localhost:9092 -t feature-stream
        ```

9.  **Verify Monitoring Integration:**
    *   **FeatureLens Metrics Endpoint:** Open `http://localhost:8081/metrics` in your browser. You should see Prometheus-formatted text metrics exposed by your running local application.
    *   **Prometheus Targets:**
        1.  Navigate to the Prometheus UI: `http://localhost:9090`.
        2.  Go to **Status -> Targets**.
        3.  Find the `featurelens` job. Its endpoint should be `http://host.docker.internal:8081/metrics`.
        4.  The **State** must be **UP**. If it's `DOWN`, re-check if `./featurelens` is running locally, listening on `:8081`, and that `prometheus.yml` targets `host.docker.internal:8081`.
    *   **Grafana Dashboard:**
        1.  Navigate to the Grafana UI: `http://localhost:3000`.
        2.  Login (default: `admin`/`admin`).
        3.  Go to **Dashboards** (folder icon in the left sidebar).
        4.  Look for a provisioned folder (e.g., `FeatureLens`) and the dashboard inside (e.g., `FeatureLens Overview`).
        5.  Open the dashboard. Once test data flows through the system, the panels should populate with visualizations based on the metrics. If the dashboard is missing, check the provisioning setup (`grafana/provisioning` directory and `docker-compose.yml` volume mounts).

10. **Stopping the Development Environment:**
    *   Stop the local FeatureLens application (usually `Ctrl+C` in the terminal where it's running).
    *   Stop the Docker Compose services:
        ```bash
        docker-compose down
        ```
        *(Add `-v` if you want to remove Kafka/Zookeeper data volumes for a completely fresh start next time: `docker-compose down -v`)*

---

*See `CONTRIBUTING.md` for a more detailed development workflow guide, including testing strategies and the pull request process.*

---

## 🗺️ Roadmap

*(Potential future enhancements and plans beyond the current scope.)*

*   **Enhanced Statistics:** Support for Min/Max, Median, Quantiles, Cardinality (for categorical features).
*   **Drift Detection Algorithms:** Implement statistical tests like Kolmogorov-Smirnov (KS test), Population Stability Index (PSI).
*   **Data Formats:** Support for Protobuf, Avro alongside JSON.
*   **Alerting Integrations:** Send alerts generated from thresholds directly to Alertmanager, Slack, PagerDuty, etc. (beyond just logging).
*   **Advanced Grafana Dashboards:** More detailed visualizations, template variables for dynamic filtering, annotations for alerts.
*   **Data Source Flexibility:** Add support for consuming from other sources like AWS Kinesis, Google Pub/Sub, or Pulsar.
*   **State Management:** Implement more robust state management for windowing, potentially using external stores for fault tolerance and scalability.
*   **Advanced Windowing:** Support for sliding windows, session windows.
*   **(Optional) Web UI:** A simple interface for configuration management, viewing current status, and recent alerts.

---

## 🙌 Contributing

We welcome contributions! Please see `CONTRIBUTING.md` for details on how to contribute, including:

*   Reporting Bugs or Requesting Features (via GitHub Issues)
*   Code Style and Linting (Pre-commit hooks using `golangci-lint` etc.)
*   Pull Request Process
*   Setting up Your Development Environment (as described above)

---

## 📄 License

Licensed under the Apache License, Version 2.0. See [LICENSE](LICENSE) for the full license text.
