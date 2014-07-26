/* jshint unused:false */
/* global console */
(function(
	Base,
	Vector,
	Logo,
	Shapes,
	Grid,
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

	shapes     : [],
	particles  : [],
	particlesA : [],
	particlesB : [],

	_deferredParticles: [],

	ticks: [],

	starGeneratorRate: 600,

	mouse: {
		x: -9999,
		y: -9999
	},

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

		this._handleMouseCoords = this._handleMouseCoords.bind(this);
		window.addEventListener('mousemove', this._handleMouseCoords, false);

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
			.wait(1000)
			.then(function(){
				this.starGeneratorRate = 200;
			}, this)
			.wait(1000)
			.then(function(){
				this.showGrid = true;
			}, this)
			.wait(2000)
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
			}, this)
			.wait(1000)
			.then(function(){
			}, this);
	},

	setupStarfield: function(){
		this.particles = [];
		// this.generateParticles(50, true);
		this.generateParticles(400);
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

		this.grid = new Engine.Shape.Puller(this.width, this.height, Grid);
	},


	getAverageTickTime: function(){
		var sum = 0, s;

		for (s = 0; s < this.ticks.length; s++) {
			sum += this.ticks[s];
		}

		console.log('Average Tick Time:', sum / this.ticks.length);
	},

	getLongestTick: function(){
		var max = 0, index, s;

		for (s = 0; s < this.ticks.length; s++) {
			if (this.ticks[s] > max) {
				max = this.ticks[s];
				index = s;
			}
		}

		console.log('Max tick was:', max, 'at index:', index);
	},

	render: function(){
		var scale = this.scale, tickStart;

		if (window.scrollY > 700) {
			window.requestAnimationFrame(this.render);
			return;
		}

		tickStart = window.performance.now();

		this.context.clearRect(
			-(this.width  / 2) * scale,
			-(this.height / 2) * scale,
			this.width  * scale,
			this.height * scale
		);

		this.now = Date.now() / 1000;

		this.tick = Math.min(this.now - this.last, 0.017);

		this.renderStarfield(this.now);

		if (this.showGrid) {
			this.grid
				.update(this)
				.draw(this.context, scale, this);
		}

		if (this.showShapes) {
			// this.renderTessellation(this.now);
			this.logo
				.update(this)
				.draw(this.context, scale, this);
		}

		this.last = this.now;

		this.ticks.push(window.performance.now() - tickStart);

		window.requestAnimationFrame(this.render);
	},

	renderTessellation: function(){
		var scale = this.scale, p, index;

		for (p = 0; p < this.shapes.length; p++)  {
			this.shapes[p]
				.update(this)
				.draw(this.context, scale, this);
		}

		this.logo
			.update(this)
			.draw(this.context, scale, this);

		// Remove destroyed shapes
		for (p = 0; p < this._deferredShapes.length; p++) {
			index = this.shapes.indexOf(this._deferredShapes.pop());
			if (index >= 0) {
				this.shapes.splice(index, 1);
			}
		}

		// 1 Per second? Maybe?
		// if (Engine.getRandomFloat(0,100) < 1.6666) {
		//     this.generateRandomShape();
		// }
	},

	generateRandomShape: function(){
		var p, index, rando, halfWidth, halfHeight, iter,
			shape, shapeTemplate, columns, rows, modWidth, row, column,
			xOffset, yOffset;

		iter = 140;

		rows = this.height / iter - 1;
		modWidth = this.width % iter;
		columns  = (this.width - modWidth) / iter - 1;

		row    = Engine.getRandomInt(0, rows);
		column = Engine.getRandomInt(0, columns);

		halfWidth  = this.width  / 2;
		halfHeight = this.height / 2;
		shapeTemplate = Shapes[Engine.getRandomInt(0, Shapes.length - 1)];

		xOffset = Engine.getRandomInt(-50, 50);
		yOffset = Engine.getRandomInt(-50, 50);

		shape = new Engine.Shape(
			(iter / 2) + (column * iter) - (modWidth / 2) - halfWidth + xOffset - 25,
			(iter / 2) + (row * iter) - halfHeight + yOffset - 25,
			50,
			50,
			shapeTemplate.points,
			shapeTemplate.polygons,
			true
		);
		shape.selfDestruct(10);
		this.shapes.push(shape);
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

		if (this.grid) {
			this.grid.resize(this.width, this.height);
		}
	},

	renderStarfield: function(){
		var scale = this.scale, p, index, particle;

		// Update all particles... may need to be optimized
		for (p = 0; p < this.particles.length; p++) {
			this.particles[p].update(this);
		}

		// Batch render particles based on color
		// to prevent unneeded context state change
		this.context.fillStyle = '#8750c2';
		for (p = 0; p < this.particlesA.length; p++) {
			particle = this.particlesA[p];

			if (particle.radius < 0.25) {
				continue;
			}
			this.context.fillRect(
				particle.pos.x * scale >> 0,
				particle.pos.y * scale >> 0,
				particle.radius * scale,
				particle.radius * scale
			);
		}

		this.context.fillStyle = '#b976ff';
		for (p = 0; p < this.particlesB.length; p++) {
			particle = this.particlesB[p];

			if (particle.radius < 0.25) {
				continue;
			}
			this.context.fillRect(
				particle.pos.x * scale >> 0,
				particle.pos.y * scale >> 0,
				particle.radius * scale,
				particle.radius * scale
			);
		}

		this.particlesA.length = 0;
		this.particlesB.length = 0;

		// Remove destroyed particles
		for (p = 0; p < this._deferredParticles.length; p++) {
			index = this.particles.indexOf(this._deferredParticles.pop());
			if (index >= 0) {
				this.particles.splice(index, 1);
			}
		}

		this.generateParticles(this.starGeneratorRate * this.tick >> 0);
	},

	_handleMouseCoords: function(event){
		this.mouse.x = event.pageX;
		this.mouse.y = event.pageY;
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
	window.Shapes,
	window.Grid,
	window.Chainable
);
