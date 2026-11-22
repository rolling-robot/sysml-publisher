package System {
    private import ScalarValues::*;
    private import SI::*;
    private import UavDefinitions::*;
    private import Views::*;
    private import StandardViewDefinitions::*;

    part <us> uavSystem {
        doc /*
        The complete unmanned air vehicle system: the UAV with its payload,
        and the ground remote control with its operator.
        */
        @PartMetadata {
            owner = "pete";
            revision = 4;
            reviewed = true;
            note = "flight-ready demonstrator";
        }
        rep language "asciidoc"
        /*
        = sysml-name:[] (sysml-short-name:[])

        sysml-doc::[]

        sysml-metadata-table::PartMetadata[]

        == Properties

        [cols="1,1"]
        |===
        |mission range | sysml-value:missionRange[]
        |operators | sysml-value:operatorCount[]
        |deployment | sysml-value:deployment[]
        |flight ready | sysml-value:flightReady[]
        |===

        == Interfaces

        sysml-members-list::InterfaceUsage[]

        == System diagram

        sysml-view::systemInterfaces[]

        == Parts

        sysml-view::systemParts[]

        == Figure

        sysml-fig::[]
        */

        part <U1> uav :> UavKinds::uav {
            doc /*
            The air vehicle: the airframe, the electric propulsion, the
            avionics, the payload bay, the CAN bus, the power line and the two
            antennas.
            */
            rep language "asciidoc"
            /*
            == Air vehicle

            [cols="1,1"]
            |===
            |serial | sysml-value:serial[]
            |autonomous | sysml-value:autonomous[]
            |===

            == Interfaces

            sysml-members-list::InterfaceUsage[]

            == Avionics diagram

            sysml-view::avionicsInterfaces[]
            */
            attribute serial : String default "UAV-001";
            attribute autonomous : Boolean default true;

            part fuselage : UavPart {
                doc /*
                The fuselage carries the avionics, the battery, the power line
                and the payload bay, and mounts the wing and the tail.
                */
                rep language "asciidoc"
                /*
                == Airframe

                [cols="1,1"]
                |===
                |length | sysml-value:length[]
                |material | sysml-value:material[]
                |watertight | sysml-value:watertight[]
                |===
                */
                attribute :>> mass = 8 [kg];
                attribute length : ISQ::LengthValue default 2.5 [m];
                attribute material : String default "carbon fiber";
                attribute watertight : Boolean default false;
            }

            part <W> wing : UavPart {
                doc /*
                The wing gives the UAV its lift. It is built in two halves
                and carries the ailerons.
                */
                rep language "asciidoc"
                /*
                == Geometry

                [cols="1,1"]
                |===
                |span | sysml-value:span[]
                |area | sysml-value:area[]
                |mean chord | sysml-value:meanChord[]
                |aspect ratio | sysml-value:aspectRatio[]
                |airfoil | sysml-value:airfoil[]
                |size | sysml-value:size[]
                |===
                */
                attribute :>> mass = 10 [kg];
                attribute span : ISQ::LengthValue default 5 [m];
                attribute area : ISQ::AreaValue default 1.5 [m^2];
                attribute meanChord : ISQ::LengthValue = area / span;
                attribute aspectRatio : Real = span * span / area;
                attribute airfoil : String default "NACA 2412";
                attribute size : Size;

                part <LW> left {
                    doc /*
                    The left wing half, with the left aileron.
                    */
                    rep language "asciidoc"
                    /*
                    = sysml-name:[] (sysml-short-name:[])

                    sysml-doc::[]

                    == Properties

                    [cols="1,1"]
                    |===
                    |mass | sysml-value:mass[]
                    |tip device | sysml-value:tipDevice[]
                    |===

                    == Figure

                    sysml-fig::[]
                    */
                    attribute mass : ISQ::MassValue default 5 [kg];
                    attribute tipDevice : String default "none";
                }

                part <RW> right {
                    doc /*
                    The right wing half, with the right aileron.
                    */
                    rep language "asciidoc"
                    /*
                    = sysml-name:[] (sysml-short-name:[])

                    sysml-doc::[]

                    == Properties

                    [cols="1,1"]
                    |===
                    |mass | sysml-value:mass[]
                    |tip device | sysml-value:tipDevice[]
                    |===

                    == Figure

                    sysml-fig::[]
                    */
                    attribute mass : ISQ::MassValue default 5 [kg];
                    attribute tipDevice : String default "none";
                }
            }

            part tail : UavPart {
                doc /*
                The tail carries the horizontal and vertical stabilisers and the
                elevator and rudder.
                */
                rep language "asciidoc"
                /*
                == Tail data

                [cols="1,1"]
                |===
                |configuration | sysml-value:configuration[]
                |area | sysml-value:area[]
                |incidence | sysml-value:incidence[]
                |===
                */
                attribute :>> mass = 2.5 [kg];
                attribute configuration : TailType default TailType::conventional;
                attribute area : ISQ::AreaValue default 0.4 [m^2];
                attribute incidence : ISQ::AngularMeasureValue default 0 ['°'];
            }

            part landingGear : UavPart {
                doc /*
                The fixed tricycle landing gear lets the UAV take off and
                land on a prepared strip.
                */
                rep language "asciidoc"
                /*
                == Gear data

                [cols="1,1"]
                |===
                |wheels | sysml-value:wheels[]
                |retractable | sysml-value:retractable[]
                |track | sysml-value:track[]
                |===
                */
                attribute :>> mass = 3 [kg];
                attribute wheels : Integer default 3;
                attribute retractable : Boolean default false;
                attribute track : ISQ::LengthValue default 1.2 [m];
            }

            part motor : Motor {
                doc /*
                The brushless motor turns the propeller; the motor controller
                sets its speed.
                */
                @PartMetadata {
                    revision = 1;
                }
                port :>> powerIn : UavLinks::PowerPort;
            }

            part propeller : Propeller {
                doc /*
                The folding propeller provides the thrust in cruise.
                */
            }

            part battery : Battery {
                doc /*
                The twelve-cell pack powers the motor and the avionics.
                */
                @PartMetadata {
                    owner = "pete";
                    revision = 2;
                    reviewed = true;
                }
                port :>> powerOut : UavLinks::PowerPort;
            }

            part powerDistribution : PowerDistribution {
                doc /*
                The power line: it splits the battery's current to the BEC and
                the motor controller.
                */
                port :>> battery : UavLinks::PowerPort;
                port :>> bec : UavLinks::PowerPort;
                port :>> motorController : UavLinks::PowerPort;
            }

            part bec : Bec {
                doc /*
                The BEC regulates the battery voltage to 5 V for the avionics
                and the payload.
                */
                port :>> powerIn : UavLinks::PowerPort;
                port :>> flightController : UavLinks::PowerPort;
                port :>> imu : UavLinks::PowerPort;
                port :>> gps : UavLinks::PowerPort;
                port :>> payloadRadio : UavLinks::PowerPort;
                port :>> payload : UavLinks::PowerPort;
            }

            part motorController : MotorController {
                doc /*
                The motor controller drives the motor from the battery current.
                */
                port :>> powerIn : UavLinks::PowerPort;
                port :>> motor : UavLinks::PowerPort;
                port :>> can : UavLinks::CanPort;
            }

            part flightController : FlightController {
                doc /*
                The flight controller stabilises the UAV and flies the
                mission.
                */
                port :>> can : UavLinks::CanPort;
                port :>> powerIn : UavLinks::PowerPort;
            }

            part imu : Imu {
                doc /*
                The IMU measures the UAV's rotation rates and
                accelerations.
                */
                port :>> can : UavLinks::CanPort;
                port :>> powerIn : UavLinks::PowerPort;
            }

            part gps : Gps {
                doc /*
                The GPS receiver fixes the UAV's position.
                */
                port :>> can : UavLinks::CanPort;
                port :>> powerIn : UavLinks::PowerPort;
            }

            part payloadBay : UavPart {
                doc /*
                The payload bay holds the mission payload and feeds it data and
                power.
                */
                rep language "asciidoc"
                /*
                == Bay data

                [cols="1,1"]
                |===
                |volume | sysml-value:volume[]
                |maximum payload | sysml-value:maxPayload[]
                |open | sysml-value:open[]
                |===
                */
                attribute :>> mass = 1.5 [kg];
                attribute volume : ISQ::VolumeValue default 8 [L];
                attribute maxPayload : ISQ::MassValue default 8 [kg];
                attribute open : Boolean default false;

                part payload : Payloads::Payload {
                    doc /*
                    The generic payload: a camera or a sensor package in the
                    bay.
                    */
                    @PartMetadata {
                        owner = "camera team";
                    }
                    port :>> data : Payloads::DataPort;
                    port :>> powerIn : UavLinks::PowerPort;
                }
            }

            part payloadRadio : Payloads::PayloadRadio {
                doc /*
                The payload radio sends the payload's data to the ground over
                the data antenna, sits on the CAN bus and draws power from the
                BEC.
                */
                port :>> rf : UavLinks::RfPort;
                port :>> data : Payloads::DataPort;
                port :>> can : UavLinks::CanPort;
                port :>> powerIn : UavLinks::PowerPort;
            }

            part commandAntenna : Antenna {
                doc /*
                The command antenna carries the pilot's control inputs.
                */
                @PartMetadata {
                    note = "spare";
                }
                port :>> rf : UavLinks::RfPort;
                attribute :>> mass = 0.1 [kg];
                attribute :>> frequency = 2400 [MHz];
                attribute :>> gain = 3;
            }

            part dataAntenna : Antenna {
                doc /*
                The data antenna carries the telemetry and payload downlink.
                */
                port :>> rf : UavLinks::RfPort;
                attribute :>> mass = 0.15 [kg];
                attribute :>> frequency = 5800 [MHz];
                attribute :>> gain = 5;
            }

            interface batteryPower connect (battery.powerOut, powerDistribution.battery);

            interface becPower connect (powerDistribution.bec, bec.powerIn);

            interface motorControllerPower connect (powerDistribution.motorController, motorController.powerIn);

            interface motorPower connect (motorController.motor, motor.powerIn);

            interface flightControllerPower connect (bec.flightController, flightController.powerIn);

            interface imuPower connect (bec.imu, imu.powerIn);

            interface gpsPower connect (bec.gps, gps.powerIn);

            interface payloadRadioPower connect (bec.payloadRadio, payloadRadio.powerIn);

            interface payloadPower connect (bec.payload, payloadBay.payload.powerIn);

            interface canBusLink connect (flightController.can, imu.can, gps.can, payloadRadio.can, motorController.can) {
                rep language "asciidoc"
                /*
                = sysml-name:[]

                The CAN bus itself, as one interface joining the nodes on it:
                the flight controller, the IMU, the GPS, the payload radio and
                the motor controller.

                == Ends

                sysml-iface-ends::[]

                == Figure

                sysml-fig::[]
                */
            }

            interface payloadDataLink connect (payloadBay.payload.data, payloadRadio.data);

            interface payloadRfLink connect (payloadRadio.rf, dataAntenna.rf);

            view avionicsInterfaces {
                expose uav;
                render asInterconnectionDiagram;
                attribute depth = 1;
                filter @PartUsage;
            }
        }

        part <RC> remoteControl {
            doc /*
            The ground remote control: two receivers and the pilot, with a
            console port for the operator link.
            */
            rep language "asciidoc"
            /*
            = sysml-name:[] (sysml-short-name:[])

            sysml-doc::[]

            == Properties

            [cols="1,1"]
            |===
            |mass | sysml-value:mass[]
            |channels | sysml-value:channels[]
            |battery life | sysml-value:batteryLife[]
            |range | sysml-value:range[]
            |screen | sysml-value:screen[]
            |===

            == Parts

            sysml-members-list::PartUsage[]

            == Interfaces

            sysml-ifaces::[]

            == Figure

            sysml-fig::[]
            */
            attribute mass : ISQ::MassValue default 2.5 [kg];
            attribute channels : Integer default 16;
            attribute batteryLife : ISQ::DurationValue default 8 [h];
            attribute range : ISQ::LengthValue default 10 [km];
            attribute screen : Boolean default true;
            port console : UavLinks::ConsolePort;

            part commandReceiver : CommandReceiver {
                doc /*
                The receiver for the command link.
                */
                port :>> rf : UavLinks::RfPort;
            }

            part dataReceiver : DataReceiver {
                doc /*
                The receiver for the data link.
                */
                port :>> rf : UavLinks::RfPort;
            }

            part <O> operator : Operator {
                doc /*
                The pilot who flies the UAV.
                */
                port :>> console : UavLinks::ConsolePort;
            }
        }

        attribute missionRange : ISQ::LengthValue = uav.endurance * uav.cruiseSpeed;
        attribute operatorCount : Integer default 1;
        attribute deployment : String default "field";
        attribute flightReady : Boolean default true;

        interface commandLink connect (uav.commandAntenna.rf, remoteControl.commandReceiver.rf) {
            rep language "asciidoc"
            /*
            = sysml-name:[]

            The command-and-control link: the pilot's inputs travel from the
            ground receiver to the UAV's command antenna.

            == Ends

            sysml-iface-ends::[]

            == Figure

            sysml-fig::[]
            */
        }

        interface dataLink connect (uav.dataAntenna.rf, remoteControl.dataReceiver.rf) {
            rep language "asciidoc"
            /*
            = sysml-name:[]

            The data link: telemetry and payload data travel between the
            UAV's data antenna and the ground receiver.

            == Ends

            sysml-iface-ends::[]

            == Figure

            sysml-fig::[]
            */
        }

        interface operatorLink connect (remoteControl.operator.console, remoteControl.console) {
            rep language "asciidoc"
            /*
            = sysml-name:[]

            The operator link: the pilot's console connects to the remote
            control.

            == Ends

            sysml-iface-ends::[]

            == Figure

            sysml-fig::[]
            */
        }

        view systemInterfaces {
            expose uavSystem;
            render asInterconnectionDiagram;
        }

        view systemParts {
            expose uavSystem::*;
            render asElementTable;
        }
    }
}
