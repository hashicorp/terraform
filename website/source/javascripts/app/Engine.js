/* jshint unused: false */
/* global console */
(function(
	Base,
	Vector,
	Logo,
	Chainable
){

var sqrt, pow, Engine;

if (!window.requestAnimationFrame) {
	window.requestAnimationFrame = (function(){
		return  window.requestAnimationFrame   ||
			window.webkitRequestAnimationFrame ||
			window.mozRequestAnimationFrame    ||
			function( callback ){
				window.setTimeout(callback, 1000 / 60);
			};
	})();
}

sqrt = Math.sqrt;
pow  = Math.pow;

Engine = Base.extend({

	scale: window.devicePixelRatio || 1,
	// scale:1,

	shapes    : [],
	particles : [],

	_deferredParticles: [],
	_deferredShapes: [],

	constructor: function(canvas, image){
		if (typeof canvas === 'string') {
			this.canvas = document.getElementById(canvas);
		} else {
			this.canvas = canvas;
		}

		if (!this.canvas.getContext) {
			return;
		}

		this.context = this.canvas.getContext('2d');

		this.resize = this.resize.bind(this);
		this.resize();
		window.addEventListener('resize', this.resize, false);

		this.setupStarfield();
		this.setupTessellation();

		this.last = Date.now() / 1000;

		this.start = this.last;

		this.render = this.render.bind(this);
		this.render();

		this.canvas.style.opacity = 1;

		this.cssAnimations(
			document.getElementById(image)
		);
	},

	cssAnimations: function(image){
		var parent = this.canvas.parentNode;

		image.style.webkitTransform = 'translate3d(0,0,0) scale(1)';
		image.style.opacity = 1;

		new Chainable()
			.wait(3000)
			.then(function(){
				parent.className += ' state-one';
			})
			.wait(150)
			.then(function(){
				parent.className += ' state-two';
			})
			.wait(150)
			.then(function(){
				parent.className += ' state-three';
			})
			.wait(500)
			.then(function(){
				parent.className += ' state-four';
			})
			.wait(100)
			.then(function(){
				this.showShapes = true;
			}, this);
	},

	setupStarfield: function(){
		this.particles = [];
		this.generateParticles(50, true);
		this.generateParticles(200);
	},

	setupTessellation: function(canvas){
		this.shapes = [];
		this.logo = new Engine.Shape(
			-(180),
			-(180),
			360,
			360,
			Logo.Points,
			Logo.Polygons
		);
	},

	render: function(){
		var scale = this.scale;

		if (window.scrollY > 700) {
			window.requestAnimationFrame(this.render);
			return;
		}

		this.context.clearRect(
			-(this.width  / 2) * scale,
			-(this.height / 2) * scale,
			this.width * scale,
			this.height * scale
		);

		this.now = Date.now() / 1000;

		this.tick = Math.min(this.now - this.last, 0.017);

		this.renderStarfield(this.now);

		if (this.showShapes) {
			this.renderTessellation(this.now);
		}

		this.last = this.now;

		window.requestAnimationFrame(this.render);
	},

	renderTessellation: function(){
		var scale = this.scale,
			p, index;

		for (p = 0; p < this.shapes.length; p++)  {
			this.shapes[p].update(this);
			this.shapes[p].draw(this.context, scale);
		}

		this.logo.update(this);
		this.logo.draw(this.context, scale);

		// Remove destroyed shapes
		for (p = 0; p < this._deferredShapes.length; p++) {
			index = this.shapes.indexOf(this._deferredShapes.pop());
			if (index >= 0) {
				this.shapes.splice(index, 1);
			}
		}
	},

	generateParticles: function(num, fixed){
		var p;

		for (p = 0; p < num; p++) {
			if (fixed) {
				this.particles.push(new Engine.Particle.Fixed(this.width, this.height));
			} else {
				this.particles.push(new Engine.Particle(this.width, this.height));
			}
		}
	},

	resize: function(){
		var scale = this.scale;

		this.width  = window.innerWidth;
		this.height = 700;

		this.canvas.width  = this.width  * scale;
		this.canvas.height = this.height * scale;

		this.context.translate(
			this.width  / 2 * scale >> 0,
			this.height / 2 * scale >> 0
		);
	},

	renderStarfield: function(){
		var scale = this.scale, p, index;

		// Update all particles... may need to be optimized
		for (p = 0; p < this.particles.length; p++) {
			this.particles[p]
				.update(this)
				.draw(this.context, scale);
		}

		// Remove destroyed particles
		for (p = 0; p < this._deferredParticles.length; p++) {
			index = this.particles.indexOf(this._deferredParticles.pop());
			if (index >= 0) {
				this.particles.splice(index, 1);
			}
		}

		this.generateParticles(200 * this.tick >> 0);
	}

});

Engine.map = function(val, istart, istop, ostart, ostop) {
	return ostart + (ostop - ostart) * ((val - istart) / (istop - istart));
};

Engine.getRandomFloat = function(min, max) {
	return Math.random() * (max - min) + min;
};

Engine.getRandomInt = function(min, max) {
	return Math.floor(Math.random() * (max - min + 1) + min);
};

Engine.clone = function(ref) {
	var clone = {}, key;
	for (key in ref) {
		clone[key] = ref[key];
	}
	return clone;
};

window.Engine = Engine;

})(
	window.Base,
	window.Vector,
	window.Logo,
	window.Chainable
);
