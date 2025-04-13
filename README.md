# FeatureLens

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8.svg)](https://golang.org/)
[![Build Status](https://img.shields.io/badge/Build-Passing-brightgreen.svg)]() <!-- TODO: Replace with actual CI badge -->
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

**FeatureLens** is a lightweight, high-performance monitoring tool designed specifically for real-time Machine Learning
feature pipelines. Built with Go, it focuses on detecting data quality issues and drift efficiently, helping you
maintain the health and reliability of your MLOps infrastructure. Think of it as a magnifying glass for your live
feature data streams.

## 🤔 Why FeatureLens?

Machine Learning models are fundamentally dependent on the quality and consistency of their input features. In
production environments, real-time feature pipelines (e.g., feeding feature stores or directly serving models) are
critical but vulnerable to silent failures:

* **Data Quality Degradation:** Unexpected null values, outliers, or data type changes can creep in due to upstream
  issues.
* **Data Drift:** The statistical distribution of live features can deviate significantly from the distributions seen
  during training, silently harming model performance (Training-Serving Skew).
* **Pipeline Latency:** Delays in feature calculation or delivery can impact downstream applications.

While various monitoring solutions exist, they can sometimes be:

* **Resource-Intensive:** Requiring significant CPU/memory, especially when monitoring numerous features or
  high-throughput streams.
* **Complex to Deploy/Manage:** Often involving heavy dependencies or intricate setups.
* **Batch-Oriented:** Not ideally suited for detecting issues in truly real-time streams *as they happen*.

**FeatureLens aims to address these challenges by leveraging Go's strengths:**

* **🚀 High Performance & Concurrency:** Go's goroutines allow FeatureLens to efficiently monitor multiple Kafka
  topics/partitions and calculate statistics concurrently with minimal overhead.
* **📉 Low Resource Footprint:** Consumes significantly fewer resources compared to many Python-based alternatives,
  making it cost-effective.
* **📦 Simple Deployment:** Distributed as a single static binary, making containerization (Docker) and deployment
  extremely straightforward, without Python environment hassles.
* **⏱️ Real-time Focus:** Designed from the ground up for low-latency processing of streaming data.

By providing timely insights into feature health, FeatureLens helps MLOps teams **detect problems early**, **reduce
debugging time**, and **increase trust** in their production ML systems.

## ✨ Features (MVP Scope)

The initial Minimum Viable Product (MVP) will focus on delivering the core value proposition with the following
features:

* **Kafka Consumer:** Consume feature data messages from specified Apache Kafka topic(s). Assumes messages are in a
  common format (e.g., JSON).
* **Real-time Statistics Calculation (Per Feature):**
    * Process messages within configurable time windows (e.g., 1-minute tumbling windows).
    * Calculate basic data quality metrics for specified feature fields:
        * **Null / Missing Value Rate:** Percentage of null or missing values.
        * **Mean (Numerical Features):** Average value within the window.
        * **Variance / Standard Deviation (Numerical Features):** Measure of data dispersion.
        * *(Future: Cardinality for categorical, min/max, etc.)*
* **Threshold-Based Alerting:**
    * Define acceptable thresholds for calculated metrics in a configuration file (
      e.g., `null_rate < 0.05`, `mean between 10 and 20`).
    * Trigger alerts when metrics violate these thresholds.
* **Simple Logging / Alert Output:**
    * Log calculated metrics periodically.
    * Log alerts to standard output (stdout) when thresholds are breached. *(Future: Integrate with Slack, PagerDuty,
      etc.)*
* **Configuration:** Load settings (Kafka brokers, topics, features to monitor, window size, thresholds) from a
  configuration file (e.g., YAML) or environment variables.

## 🏗️ Architecture (MVP)

The MVP architecture is designed to be simple and efficient, focusing on Go's strengths in stream processing and
concurrency.

```text
+------------------+      Feature Data Stream      +--------------------------------------+      Alerts / Logs
|   Apache Kafka   | ---------------------------> |        FeatureLens (Go Process)      | ---------------------> stdout / Log File
| (Topics/Brokers) |      (e.g., JSON Msgs)       |                                      |
+------------------+                              |  +---------------------------------+ |
                                                  |  | 1. Kafka Consumer               | |
                                                  |  |    (e.g., segmentio/kafka-go)   | |
                                                  |  +---------------------------------+ |
                                                  |                  |                   |
                                                  |  +---------------------------------+ |      +-----------------+
                                                  |  | 2. Message Parser/Validator     |----->|  Configuration  |
                                                  |  +---------------------------------+ |      | (YAML/Env Vars) |
                                                  |                  |                   |      +-----------------+
                                                  |  +---------------------------------+ |
                                                  |  | 3. Stats Calculator (Windowed)  | |
                                                  |  |    - Null Rate                   | |
                                                  |  |    - Mean/Variance               | |
                                                  |  +---------------------------------+ |
                                                  |                  |                   |
                                                  |  +---------------------------------+ |
                                                  |  | 4. Threshold Checker & Alerter  | |
                                                  |  +---------------------------------+ |
                                                  +--------------------------------------+

```

**Key Components:**

1. **Kafka Consumer:**
    * Utilizes a robust Go Kafka client library (e.g., `segmentio/kafka-go` or `confluent-kafka-go`).
    * Connects to specified Kafka brokers and consumes messages efficiently from target topic(s).
    * Handles partitioning and consumer group offset management automatically.

2. **Message Parser/Validator:**
    * Decodes incoming messages (MVP assumes **JSON** format) into a predefined Go struct.
    * Performs basic **validation** to ensure required fields for monitoring are present and have expected basic types.

3. **Stats Calculator (Windowed):**
    * The core engine responsible for **real-time aggregation**.
    * Calculates essential metrics (**Null Rate**, **Mean**, **Variance** for numerical features) over configurable *
      *time windows** (e.g., 1-minute tumbling windows).
    * Manages internal state efficiently for each monitored feature across active time windows.

4. **Threshold Checker & Alerter:**
    * Compares the calculated statistics from each completed window against the **user-defined thresholds** in the
      configuration.
    * If a threshold is breached, it generates a structured **alert message**.
    * For the MVP, alerts and periodic statistics summaries are logged to **standard output (stdout)**.

5. **Configuration:**
    * Loads application settings using a flexible library like `viper`.
    * Supports loading from a **configuration file** (e.g., `config.yaml`) and/or **environment variables**.
    * Settings include: Kafka connection details, topic names, features to monitor, window parameters (size, type), and
      alert thresholds for each metric.

---

## 🚀 Getting Started

There are two main ways to run FeatureLens: connecting to your existing Kafka cluster or running everything locally using Docker Compose for quick testing.

### Option 1: Connecting to Your Existing Kafka

1.  **Prerequisites:**
    *   Go (Version 1.19+ recommended)
    *   Git
    *   Access to your Apache Kafka cluster.
2.  **Clone & Build:**
    ```bash
    git clone https://github.com/<your-username>/featurelens.git # Replace with actual repo path
    cd featurelens
    go build ./cmd/featurelens
    ```
3.  **Configure:**
    *   Copy the example configuration: `cp configs/config.example.yaml config.yaml`
    *   Edit `config.yaml`:
        *   Set `kafka.brokers` to your Kafka broker addresses (e.g., `["kafka1:9092", "kafka2:9092"]`).
        *   Set `kafka.topic` to the topic you want to monitor.
        *   Configure the `features` section with the fields to monitor and their thresholds.
4.  **Run:**
    ```bash
    ./featurelens -config config.yaml
    ```

### Option 2: Running Locally with Docker Compose (Quick Test)

This option uses Docker Compose to quickly set up a local Kafka cluster and a web UI (AKHQ) for easy experimentation.

1.  **Prerequisites:**
    *   Go (Version 1.19+ recommended)
    *   Git
    *   Docker & Docker Compose
2.  **Clone & Set up Environment:**
    ```bash
    git clone https://github.com/<your-username>/featurelens.git # Replace with actual repo path
    cd featurelens
    docker-compose up -d # Starts Kafka & AKHQ UI in background
    ```
    *   Wait a moment for services to start. AKHQ UI will be available at `http://localhost:8080`.
3.  **Create Kafka Topic:**
    *   FeatureLens needs a topic to monitor. Create it via the AKHQ UI (`http://localhost:8080` -> Topics -> Create Topic) or using the command line:
        ```bash
        # Use the default topic name 'feature-stream' or change as needed
        docker-compose exec kafka kafka-topics --create --topic feature-stream --bootstrap-server kafka:9093 --partitions 1 --replication-factor 1
        ```
4.  **Build FeatureLens (and Optional Producer):**
    ```bash
    go build ./cmd/featurelens
    go build ./cmd/producer # Builds the sample data producer
    ```
5.  **Run:**
    *   **(Optional) Start Sample Producer:** This sends sample data to the `feature-stream` topic.
        ```bash
        ./producer
        ```
        *(Press Ctrl+C to stop)*
    *   **Run FeatureLens:** Use the development config which points to local Kafka.
        ```bash
        ./featurelens -config configs/config.dev.yaml # Ensure this config exists and points to localhost:9092
        ```
6.  **Stop Environment:**
    ```bash
    docker-compose down
    ```

---

*See `CONTRIBUTING.md` for a more detailed development setup guide, including pre-commit hooks.*

---

## 🗺️ Roadmap

*(Future enhancements and plans beyond the MVP.)*

* **Enhanced Statistics:** Support for Min/Max, Median, Quantiles, Cardinality (for categorical features).
* **Drift Detection Algorithms:** Implement statistical tests like Kolmogorov-Smirnov (KS test), Population Stability
  Index (PSI).
* **Data Formats:** Support for Protobuf, Avro alongside JSON.
* **Alerting Integrations:** Send alerts to Slack, PagerDuty, OpsGenie, Alertmanager.
* **Metrics Export:** Expose internal metrics and calculated statistics via a Prometheus `/metrics` endpoint for
  monitoring and visualization (e.g., with Grafana).
* **Data Source Flexibility:** Add support for consuming from other sources like AWS Kinesis, Google Pub/Sub, or Pulsar.
* **State Management:** Implement more robust state management for windowing, potentially using external stores for
  fault tolerance.
* **Advanced Windowing:** Support for sliding windows, session windows.
* **(Optional) Web UI:** A simple interface for configuration, viewing status, and recent alerts.

---

## 🙌 Contributing

We welcome contributions! Please see `CONTRIBUTING.md` (*you'll need to create this file*) for details on how to
contribute, including:

* Reporting Bugs or Requesting Features (via GitHub Issues)
* Code Style and Linting (`golangci-lint`)
* Pull Request Process
* Setting up Your Development Environment

---

## 📄 License

Licensed under the Apache License, Version 2.0. See [LICENSE](LICENSE) for the full license text.
