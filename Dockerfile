FROM golang:1.18 as build-space

COPY . /root/
WORKDIR /root/

RUN go build

FROM gcr.io/distroless/base:nonroot

COPY --from=build-space /root/kube-ingress-index /bin/kube-ingress-index

ENTRYPOINT ["/bin/kube-ingress-index"]
