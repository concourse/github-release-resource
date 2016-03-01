FROM concourse/busyboxplus:base

# satisfy go crypto/x509
RUN cat /etc/ssl/certs/*.pem > /etc/ssl/certs/ca-certificates.crt

ADD assets/check /opt/resource/check
ADD assets/in /opt/resource/in
ADD assets/out /opt/resource/out
