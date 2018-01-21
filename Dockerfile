FROM alpine

ADD ntvp /bin/
ADD data/* /data/

WORKDIR /

ENV PORT 8889
ENV HOST ""
ENV USERNAME ""
ENV PASSWORD ""

EXPOSE $PORT

CMD [ "/bin/ntvp",  "-v" ]
