package UavDefinitions {
    private import ScalarValues::*;
    private import SI::*;
    private import MeasurementReferences::*;
    private import SIPrefixes::*;
    private import UavLinks::*;

    attribute <MHz> megahertz : FrequencyUnit {
        :>> unitConversion: ConversionByPrefix {
            :>> prefix = mega;
            :>> referenceUnit = Hz;
        }
    }

    attribute def Size {
        attribute width : ISQ::LengthValue;
        attribute height : ISQ::LengthValue;
    }

    metadata def PartMetadata {
        attribute owner : String default "nobody";
        attribute revision : Integer default 0;
        attribute reviewed : Boolean default false;
        attribute process : String = "TBD";
        attribute note : String;
    }

    enum def Chemistry {
        LiPo;
        LiIon;
    }

    enum def TailType {
        conventional;
        vTail;
        flyingWing;
    }

    part def <up> UavPart {
        rep language "asciidoc"
        /*
        = sysml-name:[] (sysml-short-name:[])

        sysml-doc::[]

        sysml-metadata-table::PartMetadata[]

        == Properties

        [cols="1,1"]
        |===
        |mass | sysml-value:mass[]
        |power | sysml-value:power[]
        |===

        == Members

        sysml-members-list::PartUsage[]

        == Interfaces

        sysml-ifaces::[]

        == Figure

        sysml-fig::[]
        */
        attribute mass : ISQ::MassValue default 0.1 [kg];
        attribute power : ISQ::PowerValue default 0 [W];
    }

    part def Battery :> UavPart {
        doc /*
        The flight battery stores the energy for the electric motor and the
        avionics. It is a twelve-cell lithium polymer pack.
        */
        rep language "asciidoc"
        /*
        == Technical data

        [cols="1,1"]
        |===
        |capacity | sysml-value:capacity[]
        |voltage | sysml-value:voltage[]
        |energy | sysml-value:energy[]
        |maximum current | sysml-value:maxCurrent[]
        |specific energy | sysml-value:specificEnergy[]
        |cells | sysml-value:cells[]
        |chemistry | sysml-value:chemistry[]
        |rechargeable | sysml-value:rechargeable[]
        |===
        */
        attribute :>> mass = 8 [kg];
        attribute capacity : ISQ::ElectricChargeValue default 22 [A*h];
        attribute voltage : ISQ::ElectricPotentialValue default 50.4 [V];
        attribute energy : ISQ::EnergyValue default 1108.8 [W*h];
        attribute maxCurrent : ISQ::ElectricCurrentValue default 200 [A];
        attribute specificEnergy : ISQ::SpecificEnergyValue default 138.6 [W*h/kg];
        attribute cells : Integer default 12;
        attribute chemistry : Chemistry default Chemistry::LiPo;
        attribute rechargeable : Boolean default true;
        port powerOut : PowerPort;
    }

    part def PowerDistribution :> UavPart {
        doc /*
        The power line: it takes the battery's main current and splits it to
        the BEC, which feeds the avionics and the payload, and to the motor
        controller.
        */
        rep language "asciidoc"
        /*
        == Technical data

        [cols="1,1"]
        |===
        |maximum current | sysml-value:maxCurrent[]
        |channels | sysml-value:channels[]
        |fuse rating | sysml-value:fuseRating[]
        |===
        */
        attribute :>> mass = 0.4 [kg];
        attribute :>> power = 0.5 [W];
        attribute maxCurrent : ISQ::ElectricCurrentValue default 200 [A];
        attribute channels : Integer default 3;
        attribute fuseRating : ISQ::ElectricCurrentValue default 150 [A];
        port battery : PowerPort;
        port bec : PowerPort;
        port motorController : PowerPort;
    }

    part def Bec :> UavPart {
        doc /*
        The battery eliminator circuit regulates the battery voltage down to
        the 5 V the avionics and the payload need.
        */
        rep language "asciidoc"
        /*
        == Technical data

        [cols="1,1"]
        |===
        |output voltage | sysml-value:outputVoltage[]
        |maximum current | sysml-value:maxCurrent[]
        |efficiency | sysml-value:efficiency[]
        |===
        */
        attribute :>> mass = 0.2 [kg];
        attribute :>> power = 1 [W];
        attribute outputVoltage : ISQ::ElectricPotentialValue default 5 [V];
        attribute maxCurrent : ISQ::ElectricCurrentValue default 10 [A];
        attribute efficiency : Real default 0.9;
        port powerIn : PowerPort;
        port flightController : PowerPort;
        port imu : PowerPort;
        port gps : PowerPort;
        port payloadRadio : PowerPort;
        port payload : PowerPort;
    }

    part def MotorController :> UavPart {
        doc /*
        The motor controller turns the flight controller's commands into the
        three-phase current that drives the motor.
        */
        rep language "asciidoc"
        /*
        == Technical data

        [cols="1,1"]
        |===
        |maximum current | sysml-value:maxCurrent[]
        |continuous current | sysml-value:continuousCurrent[]
        |voltage | sysml-value:voltage[]
        |===
        */
        attribute :>> mass = 0.9 [kg];
        attribute :>> power = 2 [W];
        attribute maxCurrent : ISQ::ElectricCurrentValue default 120 [A];
        attribute continuousCurrent : ISQ::ElectricCurrentValue default 80 [A];
        attribute voltage : ISQ::ElectricPotentialValue default 50.4 [V];
        port powerIn : PowerPort;
        port motor : PowerPort;
        port can : CanPort;
    }

    part def Motor :> UavPart {
        doc /*
        The brushless electric motor drives the propeller through the speed
        controller that the flight controller commands.
        */
        rep language "asciidoc"
        /*
        == Technical data

        [cols="1,1"]
        |===
        |torque | sysml-value:torque[]
        |motor constant | sysml-value:kv[]
        |efficiency | sysml-value:efficiency[]
        |===
        */
        attribute :>> mass = 2.5 [kg];
        attribute :>> power = 4 [kW];
        attribute torque : ISQ::TorqueValue default 9.5 [N*m];
        attribute kv : Integer default 180;
        attribute efficiency : Real default 0.92;
        port powerIn : PowerPort;
    }

    part def Propeller :> UavPart {
        doc /*
        The two-blade folding propeller converts the motor's torque into
        thrust.
        */
        rep language "asciidoc"
        /*
        == Technical data

        [cols="1,1"]
        |===
        |diameter | sysml-value:diameter[]
        |pitch | sysml-value:pitch[]
        |static thrust | sysml-value:staticThrust[]
        |blades | sysml-value:blades[]
        |===
        */
        attribute :>> mass = 0.4 [kg];
        attribute diameter : ISQ::LengthValue default 610 [mm];
        attribute pitch : ISQ::LengthValue default 254 [mm];
        attribute staticThrust : ISQ::ForceValue default 150 [N];
        attribute blades : Integer default 2;
    }

    part def FlightController :> UavPart {
        doc /*
        The flight controller runs the attitude and navigation loops and
        commands the motor and the control surfaces.
        */
        rep language "asciidoc"
        /*
        == Technical data

        [cols="1,1"]
        |===
        |update rate | sysml-value:updateRate[]
        |processor | sysml-value:processor[]
        |firmware | sysml-value:firmware[]
        |===
        */
        attribute :>> mass = 0.15 [kg];
        attribute :>> power = 5 [W];
        attribute updateRate : ISQ::FrequencyValue default 400 [Hz];
        attribute processor : String default "STM32H7";
        attribute firmware : String;
        port can : CanPort;
        port powerIn : PowerPort;
    }

    part def Imu :> UavPart {
        doc /*
        The inertial measurement unit supplies angular rates and
        accelerations to the flight controller.
        */
        rep language "asciidoc"
        /*
        == Technical data

        [cols="1,1"]
        |===
        |sample rate | sysml-value:sampleRate[]
        |gyro range | sysml-value:gyroRange[]
        |calibrated | sysml-value:calibrated[]
        |===
        */
        attribute :>> mass = 0.03 [kg];
        attribute :>> power = 0.2 [W];
        attribute sampleRate : ISQ::FrequencyValue default 1000 [Hz];
        attribute gyroRange : Real default 2000;
        attribute calibrated : Boolean default false;
        port can : CanPort;
        port powerIn : PowerPort;
    }

    part def Gps :> UavPart {
        doc /*
        The GPS receiver gives the position and time the navigation loops
        need.
        */
        rep language "asciidoc"
        /*
        == Technical data

        [cols="1,1"]
        |===
        |update rate | sysml-value:updateRate[]
        |accuracy | sysml-value:accuracy[]
        |constellations | sysml-value:constellations[]
        |===
        */
        attribute :>> mass = 0.08 [kg];
        attribute :>> power = 0.5 [W];
        attribute updateRate : ISQ::FrequencyValue default 10 [Hz];
        attribute accuracy : ISQ::LengthValue default 2.5 [m];
        attribute constellations : Integer default 4;
        port can : CanPort;
        port powerIn : PowerPort;
    }

    part def CanBus :> UavPart {
        doc /*
        The CAN bus links the flight controller, the IMU, the GPS, the
        payload radio and the motor controller.
        */
        rep language "asciidoc"
        /*
        == Technical data

        [cols="1,1"]
        |===
        |bit rate | sysml-value:bitRate[]
        |nodes | sysml-value:nodes[]
        |terminated | sysml-value:terminated[]
        |===
        */
        attribute :>> mass = 0.2 [kg];
        attribute :>> power = 0.5 [W];
        attribute bitRate : ISQ::TransferRateValue default 125000 [B/s];
        attribute nodes : Integer default 5;
        attribute terminated : Boolean default true;
        port flightController : CanPort;
        port imu : CanPort;
        port gps : CanPort;
        port payloadRadio : CanPort;
        port motorController : CanPort;
    }

    part def Antenna :> UavPart {
        doc /*
        A small omnidirectional antenna for one of the radio links. The
        command link and the data link each use one.
        */
        rep language "asciidoc"
        /*
        == Technical data

        [cols="1,1"]
        |===
        |frequency | sysml-value:frequency[]
        |gain | sysml-value:gain[]
        |connector | sysml-value:connector[]
        |===
        */
        attribute :>> mass default 0.1 [kg];
        attribute frequency : ISQ::FrequencyValue default 2400 [MHz];
        attribute gain : Real default 3;
        attribute connector : String default "SMA";
        port rf : RfPort;
    }

    part def CommandReceiver :> UavPart {
        doc /*
        The command receiver on the ground station turns the pilot's inputs
        into a control stream for the flight controller.
        */
        rep language "asciidoc"
        /*
        == Technical data

        [cols="1,1"]
        |===
        |frequency | sysml-value:frequency[]
        |sensitivity | sysml-value:sensitivity[]
        |===
        */
        attribute :>> mass = 0.2 [kg];
        attribute :>> power = 1 [W];
        attribute frequency : ISQ::FrequencyValue default 2400 [MHz];
        attribute sensitivity : Real default -100;
        port rf : RfPort;
    }

    part def DataReceiver :> UavPart {
        doc /*
        The data receiver carries the telemetry and payload downlink between
        the UAV and the ground station.
        */
        rep language "asciidoc"
        /*
        == Technical data

        [cols="1,1"]
        |===
        |frequency | sysml-value:frequency[]
        |sensitivity | sysml-value:sensitivity[]
        |data rate | sysml-value:dataRate[]
        |===
        */
        attribute :>> mass = 0.3 [kg];
        attribute :>> power = 2 [W];
        attribute frequency : ISQ::FrequencyValue default 5800 [MHz];
        attribute sensitivity : Real default -95;
        attribute dataRate : ISQ::TransferRateValue default 250000 [B/s];
        port rf : RfPort;
    }

    part def Operator :> UavPart {
        doc /*
        The human operator flies the UAV from the ground station and monitors
        the payload.
        */
        rep language "asciidoc"
        /*
        == Technical data

        [cols="1,1"]
        |===
        |role | sysml-value:role[]
        |certified | sysml-value:certified[]
        |flight hours | sysml-value:flightHours[]
        |===
        */
        attribute :>> mass = 80 [kg];
        attribute role : String default "pilot";
        attribute certified : Boolean default true;
        attribute flightHours : ISQ::DurationValue default 120 [h];
        port console : ConsolePort;
    }
}
