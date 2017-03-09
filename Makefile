VERSION=0.1.0

release:
	gox  \
		-output "bin/{{.Dir}}_$(VERSION)_{{.OS}}_{{.Arch}}/{{.Dir}}" \
		 github.com/gernest/qlfu

tar:
	cd bin && ../release.sh