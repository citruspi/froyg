std.manifestYamlDoc({
    name: std.extVar('project-name'),
    maintainer: 'citruspi',
    priority: 'extra',
    section: 'default',
    version: std.join('', ['v', std.extVar('ref-version')]),
    release: std.parseInt(std.extVar('ci-job-id')),
    arch: std.extVar('arch'),
    license: 'Public Domain',
    homepage: std.extVar('project-url'),
    vendor: 'doom-fm',
    description: 'HTTP Proxy for AWS S3',
    contents: [
        { src: std.extVar('bin-name'), dst: std.join('/', ['/usr/local/bin', std.extVar('bin-name')]) },
    ],
}, indent_array_in_object=false)
