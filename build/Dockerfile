FROM ubuntu:18.04

LABEL maintainer="jungnoh.dev@gmail.com"

RUN apt-get update -y && apt-get install -y make
ADD Makefile /src/Makefile
ADD parsley-core/script /src/parsley-core/script

WORKDIR /src
RUN make ubuntu-deps
RUN make build

RUN mkdir -p /parsley/bin && mkdir -p /parsley/tmp
RUN cp /src/bin/main /parsley/bin/app
RUN cp /src/lib/core/bin/realcore /parsley/bin/core
RUN cd /parsley/bin && chmod 500 *

RUN rm -rf /src

RUN groupadd -g 801 parsley
RUN useradd -M -u 1201 -g parsley parsley
RUN chown -R parsley:parsley /parsley

WORKDIR /parsley/bin
USER parsley
ENTRYPOINT ["/parsley/bin/app"]