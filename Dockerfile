# copy to alpine image
FROM alpine
WORKDIR /app
COPY ./service /app/service
CMD ["/app/service"]
