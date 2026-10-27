FROM python:3.12-alpine
WORKDIR /app
COPY emit_graphql_spans.py .
ENV ONCE=0 INTERVAL_SEC=15 \
    SERVICE_NAME=graphql-demo SERVICE_NS=demo ENVIRONMENT=ci \
    OTLP_ENDPOINT=http://host.docker.internal:4318/v1/traces
CMD ["python3", "-u", "emit_graphql_spans.py"]
