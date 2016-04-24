(function(
	Base,
	Vector,
	Logo,
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

	constructor: function(canvas, background, tagLine){
		this.canvas     = canvas;
		this.background = background;
		this.tagLine    = tagLine;

		if (!this.canvas.getContext) {
			return null;
		}

		this.context = this.canvas.getContext('2d');

		this.setupEvents();
		this.setupStarfield();
		this.setupTessellation();
		this.setupMisc();

		this.startEngine();
	},

	startEngine: function(){
		var parent = this.canvas.parentNode;

		this.background.className += ' show';
		this.canvas.style.opacity = 1;

		// We have to pass the engine into Chainable to
		// enable the timers to properly attach to the
		// run/render loop
		new Chainable(this)
			.wait(1000)
			.then(function(){
				this.starGeneratorRate = 200;
			}, this)
			.wait(500)
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
				this.logo.startBreathing();
				this.showGrid = true;
			}, this)
			.wait(1000)
			.then(function(){
				this.typewriter.start();
			}, this);

		this.render();
	},


	setupMisc: function(){
		this.last = Date.now() / 1000;
		this.render = this.render.bind(this);

		this.typewriter = new Engine.Typewriter(this.tagLine);
	},

	setupEvents: function(){
		this.resize = this.resize.bind(this);
		this.resize();
		window.addEventListener('resize', this.resize, false);

		this._handleScroll = this._handleScroll.bind(this);
		this._handleScroll();
		window.addEventListener('scroll', this._handleScroll, false);

		this._handleMouseCoords = this._handleMouseCoords.bind(this);
		window.addEventListener('mousemove', this._handleMouseCoords, false);
	},

	setupStarfield: function(){
		this.particles = [];
		// this.generateParticles(50, true);
		this.generateParticles(400);
	},

	setupTessellation: function(canvas){
		var size, offset;
		this.shapes = [];
		if (window.innerWidth < 570) {
			size = 300;
			offset = 0;
		} else {
			size = 360;
			offset = 40;
		}

		this.logo = new Engine.Shape(
			-(size / 2),
			-(size / 2 + offset),
			size,
			size,
			Logo.points,
			Logo.polygons
		);

		this.grid = new Engine.Shape.Puller(this.width, this.height, Grid);
	},


	getAverageTickTime: function(){
		var sum = 0, s;

		for (s = 0; s < this.ticks.length; s++) {
			sum += this.ticks[s];
		}

		window.console.log('Average Tick Time:', sum / this.ticks.length);
	},

	getLongestTick: function(){
		var max = 0, index, s;

		for (s = 0; s < this.ticks.length; s++) {
			if (this.ticks[s] > max) {
				max = this.ticks[s];
				index = s;
			}
		}

		window.console.log('Max tick was:', max, 'at index:', index);
	},

	render: function(){
		var scale = this.scale, p, particle, index;

		if (this.paused) {
			return;
		}

		if (this.scrollY > this.height) {
			window.requestAnimationFrame(this.render);
			return;
		}

		this.context.clearRect(
			-(this.width  / 2) * scale,
			-(this.height / 2) * scale,
			this.width  * scale,
			this.height * scale
		);

		this.now = Date.now() / 1000;
		this.tick = Math.min(this.now - this.last, 0.017);

		// We must attach the chainable timer to the engine
		// run/render loop or else things can get pretty
		// out of wack
		if (this.updateChainTimer) {
			this.updateChainTimer(this.tick);
		}

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

		if (this.showGrid) {
			this.grid
				.update(this)
				.draw(this.context, scale, this);
		}

		if (this.showShapes) {
			this.logo
				.update(this)
				.draw(this.context, scale, this);
		}

		this.typewriter.update(this);

		this.last = this.now;

		this.generateParticles(this.starGeneratorRate * this.tick >> 0);

		window.requestAnimationFrame(this.render);
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
		var scale = this.scale,
			size, offset;

		if (window.innerWidth < 570) {
			this.height = 560;
		} else {
			this.height = 700;
		}

		this.width  = window.innerWidth;

		this.canvas.width  = this.width  * scale;
		this.canvas.height = this.height * scale;

		this.context.translate(
			this.width  / 2 * scale >> 0,
			this.height / 2 * scale >> 0
		);
		this.context.lineJoin = 'bevel';

		if (this.grid) {
			this.grid.resize(this.width, this.height);
		}

		if (this.logo) {
			if (this.height === 560) {
				size = 300;
				offset = 0;
			} else {
				size = 360;
				offset = 40;
			}
			this.logo.resize(size, offset);
		}
	},

	_handleMouseCoords: function(event){
		this.mouse.x = event.pageX;
		this.mouse.y = event.pageY;
	},

	_handleScroll: function(){
		this.scrollY = window.scrollY;
	},

	pause: function(){
		this.paused = true;
	},

	resume: function(){
		if (!this.paused) {
			return;
		}
		this.paused = false;
		this.render();
	},

	getSnapshot: function(){
		window.open(this.canvas.toDataURL('image/png'));
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
	window.Grid,
	window.Chainable
);
