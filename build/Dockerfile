FROM registry.access.redhat.com/ubi7/ubi-minimal:latest

ARG GIT_COMMIT=""
ENV OPERATOR=/usr/local/bin/snat-operator \
    USER_UID=1001 \
    USER_NAME=snat-operator

ENV GIT_COMMIT=${GIT_COMMIT}

# install operator binary
COPY build/_output/bin/snat-operator ${OPERATOR}

COPY build/bin /usr/local/bin
RUN  /usr/local/bin/user_setup

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
