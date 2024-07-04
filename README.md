# Promotion Service

A Go based application to consume and store large CSV files and access them by an endpoint.

## Table of Contents
- [Prerequisites](#prerequisites)
- [Installation](#installation)
- [Running the Application](#running-the-application)
- [API Endpoints](#api-endpoints)
- [Architecture](#architecture)
- [Technologies Used](#technologies-used)
- [Monitoring and Logging](#monitoring-and-logging)
- [Testing](#testing)
- [Contributing](#contributing)
- [License](#license)

## Prerequisites
List any prerequisites here, for example:
- Docker
- Docker Compose
- Go (1.20 or higher)

## Installation
1. Clone the repository:  
   `git clone https://github.com/sh3ll3y/promotion-service.git`
2. Navigate to the project directory:  
   `cd promotion-service`
## Running the Application
1. Build and start the services:  
   `docker-compose up --build`

2. The application will be available at `http://localhost:8080`

## API Endpoints

### Process CSV
- **Method:** `POST`
- **Endpoint:** `/process-csv`
- **Description:** Process a CSV file of promotions. 
- **Request Body:** `filename=<path-to-csv-file>`

The application users CQRS pattern and implements an efficient, parallel processing mechanism for CSV files. Once the file is uploaded, we trigger an event to notify the system to update the read database. The consumer calls the promotion service again but in prod this can be a separate service that handles only the read part of the application. 

1. **File Streaming**: The CSV file is read line-by-line using a `csv.Reader`, minimizing memory usage.

2. **Worker Pool**: A configurable pool of worker goroutines is created to process records concurrently.

3. **Producer-Consumer Model**:
    - A single goroutine reads CSV records (producer).
    - Multiple worker goroutines process these records in parallel (consumers).

4. **Channel-based Communication**:
    - CSV records are sent through a `jobs` channel to the workers.
    - A separate `errors` channel collects any errors encountered during processing.

5. **Concurrent Error Handling**:
    - Errors from all goroutines are collected in the `errors` channel.
    - The main goroutine processes these errors after all workers have finished.

6. **Graceful Shutdown**:
    - Uses `sync.WaitGroup` to ensure all workers complete before finalizing the process.
    - Channels are properly closed to prevent goroutine leaks.

7. **Scalability**: The number of worker goroutines (`workerCount`) is configurable, allowing the process to scale based on available resources.

8. **Event Publishing**: After successful processing, an event is published to notify other parts of the system (e.g., to trigger read database updates).

This approach ensures efficient CPU utilization and memory management, enabling the processing of large CSV files without loading the entire file into memory. It also provides robustness through comprehensive error handling and system notification via event publishing.
#### Example
The CSV file should be placed in the root directory of the codebase:
```bash
curl -X POST -d "filename=/app/data/promotions.csv" http://localhost:8080/process-csv
```

### Retrieve Promotion
### GET /promotions/{id}

Retrieve a specific promotion by ID.

#### Caching Mechanism

This endpoint implements a basic caching strategy to improve read performance:

1. **Cache Check**:
    - When a promotion is requested, the system first checks the Redis cache.
    - If found, the promotion is returned directly from the cache.

2. **Database Fallback**:
    - If the promotion is not in the cache, it's fetched from the database.
    - After retrieval, the promotion is stored in the cache for future requests.

3. **Cache Duration**:
    - Cached promotions have a Time-To-Live (TTL) of 1 hour.
    - After this period, the cache entry expires and will be fetched from the database on the next request.

4. **Cache Consistency**:
    - The current implementation does not automatically invalidate cache entries when promotions are updated.
    - This means that for up to 1 hour after an update, the API might serve the previous version of a promotion.

Benefits:
- Reduced database load for read operations
- Faster response times for frequently accessed promotions
- Scalability for high-read traffic scenarios

Considerations:
- There's a potential for data inconsistency for up to 1 hour after an update.
- For use cases requiring immediate consistency, consider implementing a cache invalidation strategy or reducing the TTL.

#### Example Request

```bash
curl http://localhost:8080/promotions/0006c161-b9d2-4b62-988c-c25255a20965
```

#### Example Response
```bash
{
  "id": "0006c161-b9d2-4b62-988c-c25255a20965",
  "price": 31.46,
  "expiration_date": "2018-06-24T12:50:03Z"
}
```
#### Caching in redis can be checked by running the commands below
The promotion id will be cached for 1 hour after its first request
```bash
docker-compose exec redis sh
redis-cli
GET <promotion_id>
```

## Architecture
The Promotion Service implements a CQRS pattern:
- Separate read and write databases for optimized performance 
- Kafka for event streaming between write and read services
- It uses a pull model where, upon receiving an event, the read service pulls the data from the write service (in this app both read and write service part is implemented in the same service for simplicity)
- Redis for caching frequently accessed data
- PostgreSQL for persistent storage

## Technologies Used
- Go
- PostgreSQL
- Redis
- Apache Kafka
- Docker
- Prometheus (for monitoring)
- Zap (for logging)

## Monitoring and Logging
- Application metrics can be viewed by running  `curl http://localhost:8080/metrics`
- Prometheus metrics are available at (use metrics names from the result of the above command) `http://localhost:9090`
- Application logs can be viewed using: `docker-compose logs app`

---
## Additional Questions

**1. The .csv file could be very big (billions of entries) - how would your application perform? How would you optimize it?**

Our application is designed to handle large CSV files efficiently. For files with billions of entries, we implement and propose the following optimizations:

- **Streaming and Parallel Processing (Implemented)**:
    - The CSV file is read line-by-line, avoiding loading the entire file into memory.
    - A configurable worker pool processes records concurrently, utilizing Go's goroutines.

- **Channel-based Communication (Implemented)**:
    - Uses channels for efficient, non-blocking distribution of work among workers.

- **Error Handling and Graceful Shutdown (Implemented)**:
    - Concurrent error collection and processing.
    - Uses `sync.WaitGroup` to ensure all workers complete before finalizing.

- **Asynchronous Processing (Implemented)**:
    - Uses Kafka to asynchronously populate the read database after CSV processing.

- **Batch Operations (Implemented)**:
    - Groups records for batch database insertions to reduce the number of database calls.

- **Sharding Strategy (Proposed)**:
    - Implement sharding for both write and read databases.
    - Use consistent hashing based on promotion ID to determine the shard for each record.
    - Each write DB shard would have multiple read DB replicas.

- **Consistent Hashing for Node Failure (Proposed)**:
    - Implement consistent hashing to minimize data transfer between nodes in case of node failures.

- **Further Optimizations (Proposed)**:
    - Distributed Processing: For extremely large files, consider implementing a distributed processing system (e.g., Apache Spark) to utilize multiple machines.
    - Database Optimizations: Use bulk insert operations, temporarily disable indexes during insertion and rebuild afterwards.

This approach ensures efficient CPU utilization, memory management, and scalability, enabling the processing of large CSV files with billions of entries. The sharding strategy allows for horizontal scaling of the database layer, while consistent hashing provides resilience against node failures.

**2. How would your application perform in peak periods (millions of requests per minute)? How would you optimize it?**

Our application is designed to handle high-volume traffic efficiently. Here's how we optimize for peak periods with millions of requests per minute:
- **Read Replicas**:
    - Deploy multiple read-only database replicas to distribute the query load.
    - This allows us to scale horizontally and handle increased read traffic.
- **Consistent Hashing**:
    - Implement consistent hashing to distribute requests across database nodes.
    - This minimizes data transfer when a node goes down or new nodes are added.
    - Hash the promotion ID to determine which node should handle the retrieve-promotion-id request.
- **Caching Strategy**:
    - Utilize Redis caching to serve frequently accessed data instantly.
    - Set dynamic TTL (Time-To-Live) for cached promotion IDs based on system behavior analysis.
    - This reduces database load and improves response times.
- **Load Balancing**:
    - Implement intelligent load balancing to distribute incoming requests evenly across application servers.
- **Auto-scaling**:
    - Use auto-scaling groups to dynamically adjust the number of application servers based on traffic.

**3. How would you operate this app in production (e.g. deployment, scaling, monitoring)?**
- Production Operations

    How we operate this app in production:

    #### Deployment
  - Containerization with Docker
  - Kubernetes for orchestration
  - CI/CD pipeline for automated deployments
  - Infrastructure as Code (e.g., Terraform)

  #### Scaling
  - Kubernetes Horizontal Pod Autoscaler
  - Database read replicas and sharding
  - Redis caching with auto-scaling
  - Load balancing for traffic distribution

  #### Monitoring
    - Prometheus for metrics collection
    - Grafana for visualization
    - ELK stack for centralized logging
    - Alerting system (e.g., Prometheus Alertmanager)
    - Distributed tracing (e.g., Jaeger)

  #### Performance and Reliability
  - Regular database query optimization
  - Caching strategies to reduce DB load
  - Rate limiting to prevent API abuse
  - Circuit breakers for fault tolerance
  - Backup and disaster recovery plans

  #### Security
  - Network segmentation and firewalls
  - Regular security audits

This setup ensures efficient deployment, graceful scaling, and continuous monitoring for optimal performance and reliability in production.