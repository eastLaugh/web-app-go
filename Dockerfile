FROM ubuntu

WORKDIR /app

COPY server_binary .

EXPOSE 80

CMD ["./server_binary"]