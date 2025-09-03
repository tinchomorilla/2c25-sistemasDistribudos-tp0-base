FROM python:3.9.7-slim

# Create app directory structure
WORKDIR /app

# Copy files maintaining directory structure
COPY server/main.py /app/
COPY server/common/ /app/common/
COPY server/server/ /app/server/
COPY server/protocol/ /app/protocol/
COPY server/tests/ /app/tests/

# Set Python path
ENV PYTHONPATH=/app

# Run tests
RUN python -m unittest tests/test_common.py

ENTRYPOINT ["/bin/sh"]