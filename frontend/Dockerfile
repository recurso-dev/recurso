# ---- Build stage ----
FROM node:22-alpine AS build

WORKDIR /app

COPY package.json package-lock.json ./
RUN npm ci

COPY . .

# Leave empty to use relative paths (nginx proxies /v1, /auth, etc. to the API).
# Set only if the SPA should call an absolute API origin directly.
ARG VITE_API_BASE_URL=
# Razorpay publishable key id, baked into the checkout page (optional).
ARG VITE_RAZORPAY_KEY_ID=
ENV VITE_API_BASE_URL=${VITE_API_BASE_URL} \
    VITE_RAZORPAY_KEY_ID=${VITE_RAZORPAY_KEY_ID}

RUN npm run build

# ---- Runtime stage ----
# Unprivileged nginx: runs as uid 101 (nginx), listens on 8080.
FROM nginxinc/nginx-unprivileged:alpine

# Upstream the reverse-proxy forwards API calls to; substituted into the
# config template by the stock nginx entrypoint (envsubst on templates/).
ENV API_UPSTREAM=http://api:8080

COPY nginx.conf /etc/nginx/templates/default.conf.template
COPY --from=build /app/dist /usr/share/nginx/html

# Redundant with the base image, but explicit so Dockerfile scanners can
# verify the container never runs as root.
USER 101

EXPOSE 8080
