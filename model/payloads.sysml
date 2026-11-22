package Payloads {
    private import ScalarValues::*;
    private import SI::*;
    private import UavDefinitions::*;
    private import UavLinks::*;

    port def DataPort;

    part def <PL> Payload {
        doc /*
        A generic payload that fits the UAV's payload bay. It sends its data
        over one port, draws power from the air vehicle and links to the
        payload radio.
        */
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
        |data rate | sysml-value:dataRate[]
        |===

        == Figure

        sysml-fig::[]
        */
        attribute mass : ISQ::MassValue default 3 [kg];
        attribute power : ISQ::PowerValue default 15 [W];
        attribute dataRate : ISQ::TransferRateValue default 5000 [B/s];
        port data : DataPort;
        port powerIn : PowerPort;
    }

    part def <CP> CameraPayload :> Payload {
        doc /*
        A camera payload: heavier and hungrier than the generic payload, with
        a fixed lens.
        */
        rep language "asciidoc"
        /*
        == Camera data

        [cols="1,1"]
        |===
        |resolution | sysml-value:resolution[]
        |field of view | sysml-value:fieldOfView[]
        |===
        */
        attribute :>> mass = 3.5 [kg];
        attribute :>> power = 20 [W];
        attribute resolution : Integer default 12;
        attribute fieldOfView : ISQ::AngularMeasureValue default 84 ['°'];
    }

    part def <PR> PayloadRadio :> UavPart {
        doc /*
        The payload radio carries the payload's data to the ground station
        over the data antenna. It sits on the CAN bus and draws power from
        the BEC.
        */
        rep language "asciidoc"
        /*
        == Technical data

        [cols="1,1"]
        |===
        |frequency | sysml-value:frequency[]
        |data rate | sysml-value:dataRate[]
        |encrypted | sysml-value:encrypted[]
        |===
        */
        attribute :>> mass = 0.6 [kg];
        attribute :>> power = 8 [W];
        attribute frequency : ISQ::FrequencyValue default 5800 [MHz];
        attribute dataRate : ISQ::TransferRateValue default 250000 [B/s];
        attribute encrypted : Boolean default true;
        port rf : RfPort;
        port data : DataPort;
        port can : CanPort;
        port powerIn : PowerPort;
    }
}
