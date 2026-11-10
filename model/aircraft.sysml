package Model {
    private import ScalarValues::*;
    private import SI::*;
    private import Views::*;
    private import StandardViewDefinitions::*;

    attribute def Size {
        attribute width : ISQ::LengthValue;
        attribute height : ISQ::LengthValue;
    }
    metadata def m {
        attribute owner : String default "nobody";
        attribute revision : Integer default 0;
        attribute reviewed : Boolean default false;
        attribute process : String = "TBD";
        attribute note : String;
    }
    part def <ap> AircraftPart {
        rep language "asciidoc"
            /*
            = sysml-name:[] (sysml-short-name:[])

            sysml-doc::[]

            sysml-metadata-table::m[]

            == Properties

            [cols="1,1"]
            |===
            |weight | sysml-value:mass[]
            |power | sysml-value:power[]
            |===
            */
        attribute mass : ISQ::MassValue default 12 [kg];
        attribute power : ISQ::PowerValue default 1.5 [kW];
    }
    part aircraft : ap {
        rep language "asciidoc"
        /*
        == Purpose

        sysml-doc::[]

        == Owned parts:

        sysml-members-list::PartUsage[]

        == Interfaces:

        sysml-members-list::InterfaceUsage[]

        .Wiring
        sysml-view::wiring[]
        */

        doc /*
        The purpose of the aircraft is to transport people
        */
        part <a> wing : ap {
            doc /*
            Documentation for part sysml-name:[]
            
            Owned elements:
            
            sysml-members-list::[]
            */
            @m {
                owner = "Petr";
                reviewed = true;
                note = "Work in progress";
            }
            attribute :>> mass = 9.5 [kg];
            attribute :>> power = 750 [W];
            port 'de-icing fluid';
        }
        part engine : ap {
            doc /*
            Engine provides thrust and compressed air for life-support
            system and other systems
            */
            @m {
                owner = "Sam";
                revision = 3;
            }
            in port fuel;
            out port air;
            attribute :>> mass = 400 [kg];
            attribute cruiseThrust = 900 [N];
        }
        part airConditioner : ap {
            in port air;
            attribute :>> mass := 20 [kg];
            @m {
                owner = "Bob";
                reviewed = true;
            }
        }
        part fuelTank : ap {
            out fuel;
            attribute :>> mass = 100 [kg];
            attribute capacity = 200;
        }
        interface airDuct connect (engine.air, airConditioner.air);
        interface fuelLine connect (fuelTank.fuel, engine.fuel);

        view wiring {
            expose aircraft;
            render asInterconnectionDiagram;
        }
    }
}
