# syntax=docker/dockerfile:1
FROM golang:1.25.7-alpine AS build_stage

WORKDIR /app

ARG SERVICE_NAME=media

COPY go.mod go.sum ./
RUN --mount=type=cache,id=go-mod,target=/go/pkg/mod \
    go mod download

COPY . .

RUN --mount=type=cache,id=go-mod,target=/go/pkg/mod \
    --mount=type=cache,id=go-build,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/service ./cmd/${SERVICE_NAME}

# PyTorch has no wheels for Alpine (musl); use Debian slim (glibc).
FROM python:3.12-slim AS vision_deps

ENV HF_HOME=/root/.cache/huggingface

WORKDIR /vision
RUN apt-get update && apt-get install -y --no-install-recommends \
    libgomp1 \
    && rm -rf /var/lib/apt/lists/*

COPY scripts/vision/requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt \
    && python -c "import open_clip; open_clip.create_model_and_transforms('ViT-B-32', pretrained='openai')"

FROM python:3.12-slim AS run_stage

ENV OMP_NUM_THREADS=1 \
    MKL_NUM_THREADS=1 \
    OPENBLAS_NUM_THREADS=1 \
    HF_HUB_DISABLE_TELEMETRY=1 \
    HF_HOME=/root/.cache/huggingface \
    HF_HUB_OFFLINE=1

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    ffmpeg \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=build_stage /app/service ./service
COPY --from=build_stage /app/configs ./configs
COPY --from=build_stage /app/scripts/vision ./scripts/vision
COPY --from=vision_deps /usr/local/lib/python3.12/site-packages /usr/local/lib/python3.12/site-packages
COPY --from=vision_deps /usr/local/bin /usr/local/bin
COPY --from=vision_deps /root/.cache /root/.cache

RUN chmod +x ./service ./scripts/vision/detect_capybara.py

EXPOSE 8003/tcp

ENTRYPOINT ["./service"]
