create table data(
	id              integer not null,
	Step            integer not null,
	TargetTemp      real,
	CurrentTemp     real,
	SG              real,
	PID             real,
	Power           real,

	foreign key (id) references config(id)
)
