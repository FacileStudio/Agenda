FROM oven/bun:1 AS build

WORKDIR /app

COPY package.json bun.lockb* ./

RUN bun install --ignore-scripts

COPY . .

ARG VITE_API_BASE_URL
ENV VITE_API_BASE_URL=$VITE_API_BASE_URL

RUN bunx svelte-kit sync && bun run build

FROM nginx:alpine

COPY --from=build /app/build /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80
