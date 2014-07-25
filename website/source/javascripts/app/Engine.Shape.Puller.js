(function(
	Engine,
	Point,
	Polygon,
	Vector
){

Engine.Shape.Puller = function(x, y, width, height, points, polygons){
	var i, ref, point, poly;

	this.pos  = new Vector(x, y);
	this.size = new Vector(width, height);

	ref = {};
	this.points = [];
	this.polygons = [];

	for (i = 0; i < points.length; i++) {
		point = new Point(
			points[i].id,
			points[i].x * this.size.x,
			points[i].y * this.size.y
		);
		ref[point.id] = point;
		this.points.push(point);
	}

	for (i = 0; i < polygons.length; i++) {
		poly = polygons[i];
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

	update: function(engine){
		var p;

		for (p = 0; p < this.points.length; p++)  {
			this.points[p].update(engine);
		}

		for (p = 0; p < this.polygons.length; p++) {
			this.polygons[p].update(engine);
		}

		if (this.alpha < 0.2) {
			this.alpha += 1 * engine.tick;
		}

		return this;
	},

	draw: function(ctx, scale){
		var p;

		ctx.save();
		ctx.translate(
			this.pos.x * scale >> 0,
			this.pos.y * scale >> 0
		);
		ctx.beginPath();
		for (p = 0; p < this.polygons.length; p++) {
			this.polygons[p].draw(ctx, scale);
		}
		ctx.closePath();
		ctx.lineWidth = 1 * scale;
		ctx.strokeStyle = 'rgba(108,0,243,' + this.alpha + ')';
		ctx.stroke();

		for (p = 0; p < this.points.length; p++) {
			this.points[p].draw(ctx, scale);
		}

		ctx.beginPath();
		for (p = 0; p < this.polygons.length; p++) {
			if (this.polygons[p].checkChasing()) {
				this.polygons[p].draw(ctx, scale);
			}
		}
		ctx.closePath();
		ctx.fillStyle = 'rgba(108,0,243,0.1)';
		ctx.fill();

		ctx.restore();
		return this;
	}

};

})(
	window.Engine,
	window.Engine.Point.Puller,
	window.Engine.Polygon.Puller,
	window.Vector
);
