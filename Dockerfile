FROM scratch

COPY kube-ingress-index /bin/kube-ingress-index
ENTRYPOINT ["/bin/kube-ingress-index"]
CMD [""]
