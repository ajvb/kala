'use strict'
import gulp  from 'gulp';
import gulpWatch  from 'gulp-watch';
import child  from 'child_process'
import reload  from "gulp-livereload"
import util   from 'gulp-util'
import notifier  from 'node-notifier'
import runSequence  from 'gulp-sequence'
var argv = require('yargs').argv
var server = null;





gulp.task('server:build', function () {


    if (argv.prod) {
        console.log("Compiling go binarie to run inside Docker container")
        options.env['GOOS'] = 'linux'
        options.env['CGO_ENABLED'] = '0'
    }


    let build = child.spawnSync('go', ['build', '-o', 'kala', 'main.go'],
        {
            env: {
                'PATH': process.env.PATH,
                'GOPATH': process.env.GOPATH
            }
        })
    if (build.stderr.length) {
        var lines = build.stderr.toString()
            .split('\n').filter(function (line) {
                return line.length
            });
        lines.forEach(function (element) {
            util.log(util.colors.red(
                "Error (go install):" + element
            ));
        });
        notifier.notify({
            title: 'Error (go install)',
            message: lines
        });
    }
    console.log(build.stdout.toString())
    return build;
});
gulp.task('server:spawn', function () {

    if (server)
        server.kill();

    /* Spawn application server */
    server = child.spawn('./kala', ['run'], { cwd: './' }, function (error, stdout, stderr) {
        // work with result
    });
    //console.log(server);
    /* Trigger reload upon server start */
    server.stdout.once('data', function () {
        reload.reload('/');
    });

    /* Pretty print server log output */
    server.stdout.on('data', function (data) {
        var lines = data.toString().split('\n');
        for (var l in lines)
            if (lines[l].length)
                util.log(lines[l]);
    });

    /* Print errors to stdout */
    server.stderr.on('data', function (data) {
        process.stdout.write(data.toString());
    });
});


gulp.task('watch', () => {
    gulp.start('server:build', ['server:spawn'])
    gulpWatch('cmd/**/*.go', () => {

        gulp.start('server:build', ['server:spawn'])
    });


});