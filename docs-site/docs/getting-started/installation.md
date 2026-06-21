# Installation

Recurso can be self-hosted via Docker or used via our Cloud API.

## Cloud API

If you are using Recurso Cloud, simply [sign up](https://recurso.dev/register) to get your API Keys.

## Self-Hosted (Docker)

To run Recurso on your own infrastructure:

1.  **Clone the repository**:
    ```bash
    git clone https://github.com/recur-so/recurso.git
    cd recurso
    ```

2.  **Start Services**:
    ```bash
    docker-compose up -d
    ```

3.  **Verify**:
    Visit `http://localhost:3000` to access the Dashboard.

## Client SDKs

### Node.js
```bash
npm install @recurso/recurso-node
```

### Python
```bash
pip install recurso
```

### Go
```go
go get github.com/recur-so/recurso-go
```
