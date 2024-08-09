FROM gcr.io/distroless/static-debian11:nonroot
ENTRYPOINT ["/baton-auth0"]
COPY baton-auth0 /