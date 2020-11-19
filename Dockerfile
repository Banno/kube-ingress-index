FROM golang:1.15 as build-space

COPY . /root/
WORKDIR /root/

RUN go build

FROM scratch
COPY --from=build-space /root/kube-ingress-index /bin/kube-ingress-index

ENTRYPOINT ["/bin/kube-ingress-index"]
CMD [""]
