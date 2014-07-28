(function(
	Engine,
	Point,
	Polygon,
	Vector
){

Engine.Shape.Puller = function(width, height, json){
	var i, ref, point, poly;

	this.pos  = new Vector(0, 0);
	this.size = new Vector(width, height);
	this.heightRatio = json.data.width / json.data.height;
	this.widthRatio  = json.data.ar;

	this.resize(width, height, true);

	ref = {};
	this.points = [];
	this.polygons = [];

	for (i = 0; i < json.points.length; i++) {
		point = new Point(
			json.points[i].id,
			json.points[i].x,
			json.points[i].y,
			this.size
		);
		ref[point.id] = point;
		this.points.push(point);
	}

	for (i = 0; i < json.polygons.length; i++) {
		poly = json.polygons[i];
		this.polygons.push(new Polygon(
			ref[poly.points[0]],
			ref[poly.points[1]],
			ref[poly.points[2]],
			poly.color
		));
		this.polygons[this.polygons.length - 1].noFill = true;
	}

	this.ref = undefined;
};

Engine.Shape.Puller.prototype = {

	alpha: 0,

	sizeOffset: 100,

	resize: function(width, height, sizeOnly){
		var len, p, newWidth, newHeight;

		newHeight = height + this.sizeOffset;
		newWidth  = this.size.y * this.heightRatio;

		if (newWidth < width) {
			newWidth  = width    + this.sizeOffset;
			newHeight = newWidth * this.widthRatio;
		}

		this.size.y = newHeight;
		this.size.x = newWidth;

		this.pos.x = -(newWidth  / 2);
		this.pos.y = -(newHeight / 2);

		if (sizeOnly) {
			return this;
		}

		for (p = 0, len = this.points.length; p < len; p++) {
			this.points[p].resize();
		}
	},

	update: function(engine){
		var p;

		for (p = 0; p < this.points.length; p++)  {
			this.points[p].update(engine);
		}

		if (this.alpha < 1) {
			this.alpha = Math.min(this.alpha + 2 * engine.tick, 1);
		}

		return this;
	},

	draw: function(ctx, scale, engine){
		var p, poly;

		ctx.translate(
			this.pos.x * scale >> 0,
			this.pos.y * scale >> 0
		);

		if (this.alpha < 1) {
			ctx.globalAlpha = this.alpha;
		}

		ctx.beginPath();
		for (p = 0; p < this.polygons.length; p++) {
			poly = this.polygons[p];
			ctx.moveTo(
				poly.a.pos.x * scale >> 0,
				poly.a.pos.y * scale >> 0
			);
			ctx.lineTo(
				poly.b.pos.x * scale >> 0,
				poly.b.pos.y * scale >> 0
			);
			ctx.lineTo(
				poly.c.pos.x * scale >> 0,
				poly.c.pos.y * scale >> 0
			);
			ctx.lineTo(
				poly.a.pos.x * scale >> 0,
				poly.a.pos.y * scale >> 0
			);
		}
		ctx.closePath();
		ctx.lineWidth = 0.4 * scale;
		ctx.strokeStyle = 'rgba(108,0,243,0.15)';
		ctx.stroke();

		if (this.alpha < 1) {
			ctx.globalAlpha = 1;
		}

		for (p = 0; p < this.points.length; p++) {
			this.points[p].draw(ctx, scale);
		}

		ctx.beginPath();
		for (p = 0; p < this.polygons.length; p++) {
			if (this.polygons[p].checkChasing()) {
				poly = this.polygons[p];
				ctx.moveTo(
					poly.a.pos.x * scale >> 0,
					poly.a.pos.y * scale >> 0
				);
				ctx.lineTo(
					poly.b.pos.x * scale >> 0,
					poly.b.pos.y * scale >> 0
				);
				ctx.lineTo(
					poly.c.pos.x * scale >> 0,
					poly.c.pos.y * scale >> 0
				);
				ctx.lineTo(
					poly.a.pos.x * scale >> 0,
					poly.a.pos.y * scale >> 0
				);
			}
		}
		ctx.closePath();
		ctx.fillStyle = 'rgba(108,0,243,0.05)';
		ctx.fill();

		ctx.setTransform(1, 0, 0, 1, 0, 0);
		ctx.translate(
			engine.width  / 2 * engine.scale >> 0,
			engine.height / 2 * engine.scale >> 0
		);
		return this;
	}

};

})(
	window.Engine,
	window.Engine.Point.Puller,
	window.Engine.Polygon.Puller,
	window.Vector
);
