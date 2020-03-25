FROM debian:stretch-slim
RUN apt update && apt install -y procps ca-certificates netcat && apt clean
RUN apt install -y python3
COPY dist/ /root/
WORKDIR /root/

CMD ["python3", "-m", "http.server", "8080"]
