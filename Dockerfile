FROM scratch

COPY ./dist/linux_amd64/kube-ingress-index /bin/kube-ingress-index
ENTRYPOINT ["/bin/kube-ingress-index"]
CMD [""]
