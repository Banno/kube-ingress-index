FROM arm64v8/golang as build-space

COPY . /root/
WORKDIR /root/

RUN go build

FROM arm64v8/debian
COPY --from=build-space /root/kube-ingress-index /bin/kube-ingress-index

ENTRYPOINT ["/bin/kube-ingress-index"]
CMD [""]
