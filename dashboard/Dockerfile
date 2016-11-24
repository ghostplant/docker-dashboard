FROM docker:1.12.4
MAINTAINER CUI Wei <ghostplant@qq.com>

RUN apk update && apk add iptables nginx && rm -r /var/cache/apk
ADD dashboard /opt/dashboard

CMD ["sh", "-c", "dockerd & /opt/dashboard/run"]
