(function(Sidebar, CubeDraw){

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

	initializeHomepage: function(){
		new CubeDraw();
	},

	Pages: {
		'page-home': function(){
			Init.initializeHomepage();
		}
	}

};

Init.start();

})(window.Sidebar, window.CubeDraw);
