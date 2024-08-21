FROM denoland/deno
LABEL org.opencontainers.image.source https://github.com/waktaplay/youtubei-proxy-with-potoken

COPY . /app
WORKDIR /app

RUN deno cache index.ts

EXPOSE 8000
CMD ["run", "--config", "tsconfig.json", "--allow-read", "--allow-env", "--allow-net", "index.ts"]