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

	Pages: {
		'page-home': function(){
			var jumbotron = document.getElementById('jumbotron'),
				canvas;

			if (!jumbotron) {
				return;
			}

			canvas = document.createElement('canvas');
			canvas.className = 'terraform-canvas';

			jumbotron.appendChild(canvas);
			window.engine = new Engine(canvas, 'image');
		}
	}

};

Init.start();

})(window.Engine);
