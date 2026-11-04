// Fixture for the inline macro sysml-value:<attribute>[], which shows the
// value of an attribute of the page's element, units included. The // notes
// give the expected text on each page.
package Values {
    private import SI::*;
    private import ScalarValues::*;

    enum def Level { low; high; }

    attribute def Position {
        attribute x : Real;
        attribute y : Real;
    }

    part def Motor {
        attribute power : ISQ::PowerValue default 750 [W];
    }

    part def Pump {
        rep language "asciidoc"
            /*
            mass: sysml-value:mass[]

            volume flow: sysml-value:volumeFlow[]

            speed: sysml-value:speed[]

            top speed: sysml-value:topSpeed[]

            count: sysml-value:count[]

            ratio: sysml-value:ratio[]

            label: sysml-value:label[]

            ok: sysml-value:ok[]

            level: sysml-value:level[]

            position: sysml-value:position[]

            note: sysml-value:note[]

            motor power: sysml-value:motor.power[]
            */
        attribute mass : ISQ::MassValue default 5 [kg];
        attribute volumeFlow : ISQ::VolumeFlowRateValue default 2.5 [m^3/s];
        attribute speed : ISQ::SpeedValue default 90 [km/h];
        attribute topSpeed : ISQ::SpeedValue default 10 [km/h] * 2;
        attribute count : Integer default 3;
        attribute ratio : Real default 0.25;
        attribute label : String default "P-1";
        attribute ok : Boolean default true;
        attribute level : Level default Level::low;
        attribute position : Position;
        attribute note : String;
        part motor : Motor;
    }

    // Every value from Pump's defaults.
    // mass: 5 kg, volume flow: 2.5 m³⋅s⁻¹, speed: 90 km/h,
    // top speed: 5.555555555555555 m/s (the engine computes in m/s),
    // count: 3, ratio: 0.25, label: P-1, ok: true, level: Level::low,
    // position: [composite] and a warning, note: (no value),
    // motor power: 750 W
    part typical : Pump;

    // Its own values, redefining Pump's.
    // mass: 7.5 kg, label: P-2, level: Level::high, count: 4,
    // motor power: 1.1 kW; the rest as for typical.
    part custom : Pump {
        attribute :>> mass = 7.5 [kg];
        attribute :>> label = "P-2";
        attribute :>> level = Level::high;
        attribute :>> count = 4;
        part :>> motor {
            attribute :>> power = 1.1 [kW];
        }
    }
}
