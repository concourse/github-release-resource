FROM concourse/buildroot:base

ADD assets/check /opt/resource/check
ADD assets/in /opt/resource/in
ADD assets/out /opt/resource/out
