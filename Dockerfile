FROM scratch

COPY dist/linux_amd64/kube-ingress-ingex /bin/kube-ingress-ingex
ENTRYPOINT ["/bin/kube-ingress-ingex"]
CMD [""]
