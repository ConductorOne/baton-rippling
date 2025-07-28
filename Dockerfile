FROM gcr.io/distroless/static-debian11:nonroot
ENTRYPOINT ["/baton-rippling"]
COPY baton-rippling /