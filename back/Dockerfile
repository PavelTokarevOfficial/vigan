# syntax=docker/dockerfile:1.7
FROM golang:1.24-bookworm AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/api ./cmd/api && CGO_ENABLED=0 go build -o /out/worker ./cmd/worker
FROM debian:bookworm AS whisper-build
RUN --mount=type=cache,target=/var/cache/apt,sharing=locked apt-get update && apt-get install -y --no-install-recommends git cmake g++ make ca-certificates && rm -rf /var/lib/apt/lists/*
RUN git clone --depth=1 https://github.com/ggerganov/whisper.cpp.git /src && cmake -S /src -B /src/build -DWHISPER_BUILD_EXAMPLES=ON -DGGML_NATIVE=OFF -DGGML_CPU_ALL_VARIANTS=OFF && cmake --build /src/build -j2
FROM debian:bookworm-slim
RUN --mount=type=cache,target=/var/cache/apt,sharing=locked apt-get update && apt-get install -y --no-install-recommends ffmpeg chromium ca-certificates && ffmpeg -hide_banner -filters | grep -q 'subtitles' && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=build /out/api /out/worker ./
COPY --from=whisper-build /src/build/bin/ /opt/whisper/
RUN ln -s /opt/whisper/whisper-cli /usr/local/bin/whisper-cli && ldconfig /opt/whisper
COPY migrations ./migrations
