create table config(
    id 		            integer primary key autoincrement,
    FermenterSensor     text not null,
    PresenceZero        integer not null,
    PresenceCalibration real not null,
    PresenceOnTimer     integer not null,
    PresenceEnabled     integer not null,
    PresenceTimeout     integer not null,
    TemperatureScale    integer not null,
    TargetTemperature   real not null,
    PidSlope            real not null,
    NpaZero             integer not null,
    NpaCalibration      real not null,
    Tec1Threshold       integer not null,
    Tec1Min             integer not null,
    Tec1Max             integer not null,
    Tec2Threshold       integer not null,
    Tec2Min             integer not null,
    Tec2Max             integer not null,
    Fan1Threshold       integer not null,
    Fan1Min             integer not null,
    Fan1Max             integer not null,
    Fan2Threshold       integer not null,
    Fan2Min             integer not null,
    Fan2Max             integer not null,
    Pump1Threshold      integer not null,
    Pump1Min            integer not null,
    Pump1Max            integer not null,
    Pump2Threshold      integer not null,
    Pump2Min            integer not null,
    Pump2Max            integer not null,
    NpaMinValue         real not null,
    NpaMaxValue         real not null,
    NpaMinPressure      real not null,
    NpaMaxPressure      real not null,
    Stage               integer not null,
    OG 		            real not null,
    BrewingStartTime    date not null,
    PitchTime	        date not null
)
