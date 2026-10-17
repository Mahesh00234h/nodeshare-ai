FROM python:3.12-slim

WORKDIR /app

# Set PYTHONPATH so python can resolve gRPC generated modules directly
ENV PYTHONPATH=/app:/app/app/grpc/generated

# Install system dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    curl \
    && rm -rf /var/lib/apt/lists/*

# Copy requirements and install dependencies
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy application code
COPY . .

# Generate gRPC stubs if proto changes
RUN python -m grpc_tools.protoc -Iproto --python_out=app/grpc/generated --grpc_python_out=app/grpc/generated proto/node_controller.proto

EXPOSE 10000 50051

CMD ["sh", "-c", "uvicorn app.main:app --host 0.0.0.0 --port ${PORT:-10000}"]
