FROM litellm/litellm:latest

COPY requirements.txt /app/requirements.txt
RUN pip install --no-cache-dir -r /app/requirements.txt

COPY config.yaml /app/config.yaml
COPY pii_middleware.py /app/custom_callbacks.py

ENV LITELLM_CONFIG_PATH=/app/config.yaml
ENV LITELLM_LOG=INFO

EXPOSE 4000

ENTRYPOINT ["litellm", "--config", "/app/config.yaml", "--port", "4000"]
