/* jshint unused: false */
/* global console */
(function(Base, Vector, Circle){

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

	particles : [],
	_deferred : [],

	// points   : [],
	// polygons : [],
	shapes: [],

	speed: 1,
	accel: 0.08,

	constructor: function(canvas, bg){
		var image, el;
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

		this.setupStarfield(bg);
		this.setupTessellation();

		this.last = Date.now() / 1000;

		this.start = this.last;

		this.render = this.render.bind(this);
		this.render();

		this.canvas.style.opacity = 1;

		image = document.getElementById(bg);
		image.style.webkitTransform = 'translate3d(0,0,0) scale(1)';
		image.style.opacity = 1;

		el = document.body;

		setTimeout(function() {
			el.className += ' state-one';
			setTimeout(function() {
				el.className += ' state-two';
				setTimeout(function() {
					el.className += ' state-three';
					setTimeout(function() {
						el.className += ' state-four';
					}, 550);
				}, 200);
			}, 200);
		}, 4000);
	},

	setupStarfield: function(){
		this.particles = [];
		this.generateParticles(50, true);
		this.generateParticles(200);
	},

	setupTessellation: function(canvas){
		this.shapes = [];
		this.logo = new Engine.Shape(
			(this.width  / 2 - 180),
			(this.height / 2 - 180),
			360,
			360,
			Circle.Points,
			Circle.Polygons
		);
	},

	render: function(){
		var tick;

		if (window.scrollY > 700) {
			window.requestAnimationFrame(this.render);
			return;
		}

		// this.context.clearRect(
		//     0,
		//     0,
		//     this.width  * this.scale,
		//     this.height * this.scale
		// );

		// Potentially more performant than clearRect
		this.canvas.width = this.width * this.scale;
		this.canvas.height = 700 * this.scale;

		this.now = Date.now() / 1000;

		tick = Math.min(this.now - this.last, 0.017);
		this.tick = this.speed * tick;

		this.renderStarfield(this.now);
		this.tick = tick;

		if (this.now - this.start > 3) {
			this.renderTessellation(this.now);
		}

		this.last = this.now;

		window.requestAnimationFrame(this.render);
	},

	renderTessellation: function(){
		var scale = this.scale,
			p;

		for (p = 0; p < this.shapes.length; p++)  {
			this.shapes[p].update(this);
			this.shapes[p].draw(this.context, scale);
		}

		this.logo.update(this);
		this.logo.draw(this.context, scale);
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
		this.width  = window.innerWidth;
		this.height = 700;

		this.canvas.width  = this.width  * this.scale;
		this.canvas.height = this.height * this.scale;
	},

	renderStarfield: function(){
		var scale = this.scale, p, index;

		// Update all particles... may need to be optimized
		for (p = 0; p < this.particles.length; p++) {
			this.particles[p]
				.update(this)
				.draw(this.context, scale);
		}

		// Remove destroyed entities
		for (p = 0; p < this._deferred.length; p++) {
			index = this.particles.indexOf(this._deferred.pop());
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

})(window.Base, window.Vector, window.Circle);
