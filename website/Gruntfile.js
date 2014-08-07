// jshint node:true
module.exports = function(grunt) {

// Load plugins here
grunt.loadNpmTasks('grunt-contrib-less');
grunt.loadNpmTasks('grunt-contrib-clean');
grunt.loadNpmTasks('grunt-contrib-connect');
grunt.loadNpmTasks('grunt-contrib-copy');
grunt.loadNpmTasks('grunt-contrib-watch');
grunt.loadNpmTasks('grunt-recess');

// Configuration goes here
grunt.initConfig({

	less: {
		development:{
			files: {
				"source/stylesheets/main.css": "source/stylesheets/main.less"
			}
		}
	},


	watch: {
		less: {
			files: 'source/stylesheets/*.less',
			tasks: ['less']
		}
	}

});

// CSS Compilation task
grunt.registerTask('default', ['watch']);

};
