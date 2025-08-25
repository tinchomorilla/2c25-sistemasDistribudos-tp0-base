FROM python:3.9.7-slim
COPY server/main.py /main.py
COPY server/common /common
COPY server/tests /tests
RUN python -m unittest tests/test_common.py

RUN apt-get update && apt-get install -y netcat-openbsd && rm -rf /var/lib/apt/lists/*

ENTRYPOINT ["/bin/sh"]