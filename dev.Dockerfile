FROM scratch

COPY ./build/dnshortcut /
ENTRYPOINT [ "/dnshortcut" ]
