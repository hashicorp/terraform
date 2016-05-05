(function(
	Engine
){

// Quick and dirty IE detection
var isIE = (function(){
	if (window.navigator.userAgent.match('Trident')) {
		return true;
	} else {
		return false;
	}
})();

// isIE = true;

var Init = {

	start: function(){
		var id = document.body.id.toLowerCase();

		if (this.Pages[id]) {
			this.Pages[id]();
		}
		//always init sidebar
		Init.initializeSidebar();
	},

	initializeSidebar: function(){
		new Sidebar();
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

	initializeEngine: function(){
		var jumbotron = document.getElementById('jumbotron'),
			content   = document.getElementById('jumbotron-content'),
			tagLine   = document.getElementById('tag-line'),
			canvas, galaxy;

		if (!jumbotron) {
			return;
		}

		galaxy = document.createElement('div');
		galaxy.id = 'galaxy-bg';
		galaxy.className = 'galaxy-bg';
		jumbotron.appendChild(galaxy);

		content.appendChild(
			Init.generateAnimatedLogo()
		);

		canvas = document.createElement('canvas');
		canvas.className = 'terraform-canvas';

		jumbotron.appendChild(canvas);
		new Engine(canvas, galaxy, tagLine);
	},

	Pages: {
		'page-home': function(){
			if (isIE) {
				document.getElementById('jumbotron').className += ' static';
				document.getElementById('tag-line').style.visibility = 'visible';
				return;
			}

			Init.initializeEngine();
		}
	}

};

Init.start();

})(window.Engine);
