(function(
	Engine
){

var Init = {

	start: function(){
		var id = document.body.id.toLowerCase();

		if (this.Pages[id]) {
			this.Pages[id]();
		}
	},

	generateAnimatedLogo: function(){
		var container, x, block;

		container = document.createElement('div');
		container.className = 'animated-logo';

		for (x = 1; x < 5; x++) {
			block = document.createElement('div');
			block.className = 'white-block block-' + x;
			container.appendChild(block);
		}

		return container;
	},

	Pages: {
		'page-home': function(){
			var jumbotron = document.getElementById('jumbotron'),
				content   = document.getElementById('jumbotron-content'),
				galaxy    = document.getElementById('galaxy-bg'),
				tagLine   = document.getElementById('tag-line'),
				canvas;

			if (!jumbotron) {
				return;
			}

			content.appendChild(
				Init.generateAnimatedLogo()
			);

			canvas = document.createElement('canvas');
			canvas.className = 'terraform-canvas';

			jumbotron.appendChild(canvas);
			window.engine = new Engine(canvas, galaxy, tagLine);
		}
	}

};

Init.start();

})(window.Engine);
