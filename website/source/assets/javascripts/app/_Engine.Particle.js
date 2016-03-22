(function(
	Engine,
	Vector
){

Engine.Particle = function(width, height){
	var side, targetX, targetY;
	this.accel = Vector.coerce(this.accel);
	this.vel   = Vector.coerce(this.vel);
	this.pos   = new Vector(0, 0);

	this.maxRadius = Engine.getRandomFloat(0.1, 2.5);
	// this.maxSpeed = Engine.getRandomFloat(0.01, 1000);
	this.maxSpeed = Engine.getRandomFloat(20, 1000);

	// Pick a random target
	side = Engine.getRandomInt(0, 3);
	if (side === 0 || side === 2) {
		targetY = (side === 0) ? -(height / 2) : (height / 2);
		targetX = Engine.getRandomInt(-(width / 2), width / 2);
	} else {
		targetY = Engine.getRandomInt(-(height / 2), height / 2);
		targetX = (side === 3) ? -(width / 2) : (width / 2);
	}

	this.target = new Vector(targetX, targetY);
	this.getAccelVector();

	this.maxDistance = this.distanceTo(this.target);

	this.fillA = '#8750c2';
	this.fillB = '#b976ff';
	this.frameMax = Engine.getRandomInt(1, 5);
};

Engine.Particle.prototype = {

	radius: 1,

	frame: 0,
	showA: false,

	accel: {
		x: 0,
		y: 0
	},

	vel: {
		x: 0,
		y: 0
	},

	pos: {
		x: 0,
		y: 0
	},

	opacity: 1,

	maxSpeed: 1500,
	maxForce: 1500,

	getAccelVector: function(){
		this.accel = Vector.sub(this.target, this.pos)
			.normalize()
			.mult(this.maxSpeed);
	},

	update: function(engine){
		var distancePercent, halfWidth, halfHeight;

		this.vel
			.add(this.accel)
			.limit(this.maxSpeed);

		this.pos.add(Vector.mult(this.vel, engine.tick));

		halfWidth  = engine.width  / 2 + this.maxRadius;
		halfHeight = engine.height / 2 + this.maxRadius;

		if (
			this.pos.x < -(halfWidth) ||
			this.pos.x > halfWidth ||
			this.pos.y < -(halfHeight) ||
			this.pos.y > halfHeight
		) {
			this.kill(engine);
		}

		distancePercent = (this.maxDistance - this.distanceTo(this.target)) / this.maxDistance;
		this.radius = Math.max(0.1, this.maxRadius * distancePercent);

		this.frame++;
		if (this.frame > this.frameMax) {
			this.frame = 0;
			this.showA = !this.showA;
		}

		if (this.showA) {
			engine.particlesA[engine.particlesA.length] = this;
		} else {
			engine.particlesB[engine.particlesB.length] = this;
		}

		return this;
	},

	draw: function(ctx, scale){
		if (this.radius < 0.25) {
			return;
		}

		if (this.showA) {
			ctx.fillStyle = this.fillA;
		} else {
			ctx.fillStyle = this.fillB;
		}

		// Draw a square - very performant
		ctx.fillRect(
			this.pos.x * scale >> 0,
			this.pos.y * scale >> 0,
			this.radius * scale,
			this.radius * scale
		);

		// Draw a circle - far less performant
		// ctx.beginPath();
		// ctx.arc(
		//     this.pos.x * scale,
		//     this.pos.y * scale,
		//     this.radius * scale,
		//     0,
		//     Math.PI * 2,
		//     false
		// );
		// ctx.fill();

		return this;
	},

	kill: function(engine){
		engine._deferredParticles.push(this);
		return this;
	},

	distanceTo: function(target) {
		var xd = this.pos.x - target.x;
		var yd = this.pos.y - target.y;
		return Math.sqrt(xd * xd + yd * yd );
	}
};

})(window.Engine, window.Vector);
