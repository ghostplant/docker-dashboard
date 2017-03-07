FROM alpine:3.4
MAINTAINER CUI Wei <ghostplant@qq.com>

RUN apk --update add nginx && rm -r /var/cache/apk
ADD dashboard /opt/dashboard

EXPOSE 80
# VOLUME ["/run"]
CMD ["sh", "-c", "/opt/dashboard/run"]
