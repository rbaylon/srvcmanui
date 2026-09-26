app=srvcmanui
distdir=/usr/local/arkgate/${app}

build:
	go mod tidy
	go build -o ${app}

rc:
	install -m 755 rc.${app} /etc/rc.d/${app}
	echo "${app} install in rc.d"
	echo "use rcctl enable|start ${app} to enable and start."

install:
	mkdir -p ${distdir}
	install -m 755 ${app} ${distdir}/
	cp -r layouts ${distdir}/
	cp -r static ${distdir}/
	cp -r modules ${distdir}/
	cp utils/*.html ${distdir}/
	rm -rf ${distdir}/modules/*/routes

dist:
	make build
	make install
	cp rc.${app} ${distdir}/
	tar -C /usr/local/arkgate/ -czvf /usr/local/arkgate/${app}.tar.gz ${app}
	ls -l /usr/local/arkgate/${app}.tar.gz

clean:
	rm -rf ${distdir}
	rm -f ${app}
	rm -f /etc/rc.d/${app}
