package Model {
    private import ScalarValues::*;
    private import SI::*;

    attribute def Size {
        attribute width : ISQ::LengthValue;
        attribute height : ISQ::LengthValue;
    }

    metadata def m {
        attribute a default "a";
        attribute owner : String default "nobody";
        attribute revision : Integer default 1;
        attribute reviewed : Boolean default false;
        attribute process : String = "ISO 15288";
        attribute note : String;
    }
    part def pd {
        rep language "asciidoc"
            /*
            = sysml-short-name:[]; sysml-name:[]
            
            sysml-doc::[]

            sysml-metadata::m[]

            == Properties

            [cols="1,1"]
            |===
            |mass | sysml-value:mass[]
            |power | sysml-value:power[]
            |volume flow | sysml-value:volumeFlow[]
            |operating temperature | sysml-value:temperature[]
            |size | sysml-value:size[]
            |===
            */
        attribute mass : ISQ::MassValue default 12 [kg];
        attribute power : ISQ::PowerValue default 1.5 [kW];
        attribute volumeFlow : ISQ::VolumeFlowRateValue default 50 [L/min];
        attribute temperature : ISQ::ThermodynamicTemperatureValue default 353.15 [K];
        attribute size : Size;
    }
        part <a> aaaa : pd {
            doc /*
            Documentation for part sysml-name:[]

            Owned elements:

            sysml-members-list::[]
            */
            @m {
                a := "aa";
                owner = "pete";
                reviewed = true;
            }
            attribute :>> mass = 9.5 [kg];
            attribute :>> power = 750 [W];
            port p;
            port p1;
            port p2;
        }
        #m part <b> bbb : pd {
            doc /* This is sysml-name:[] part */
            port p;
        }
        interface connect (a.p, b.p);
    }
