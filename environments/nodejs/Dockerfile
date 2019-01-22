FROM zhangjyr/openfaas-ics:b1 AS ics

# A docker image for the func container.

# default variant is the official alpine node image (much smaller than the standard image)
FROM node:8-alpine

ARG NODE_ENV
ENV NODE_ENV $NODE_ENV

RUN mkdir -p /usr/src/app
WORKDIR /usr/src/app

COPY package.json /usr/src/app/
RUN npm install && npm cache clean --force
COPY server.js /usr/src/app/server.js

CMD [ "npm", "start" ]

RUN mkdir -p /tmp
COPY --from=ics /fwatchdog  /

ENV fprocess="node server.js --port=%d"

HEALTHCHECK --interval=5s CMD [ -e /tmp/.lock ] || exit 1

CMD ["/fwatchdog"]

EXPOSE 8080
EXPOSE 8079
