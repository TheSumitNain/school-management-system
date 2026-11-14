import { useEffect, useState, type FormEvent } from "react";
import { Modal } from "../../../shared/components/Modal";
import { Select } from "../../../shared/components/Select";
import type { StudentProfile } from "../api";
import {
  useCreateSchoolStudentLocationMutation,
  useSaveSchoolStudentAddressMutation,
  useSchoolStudentAddressesQuery,
  useSchoolStudentLocationsQuery,
} from "../queries";

type AddressType = "current" | "permanent";

type Props = {
  student: StudentProfile;
  onClose: () => void;
};

function textValue(value?: string | null) {
  return value ?? "";
}

export function StudentAddressSection({ student, onClose }: Props) {
  const [addressType, setAddressType] = useState<AddressType>("current");
  const [line1, setLine1] = useState("");
  const [line2, setLine2] = useState("");
  const [areaMohalla, setAreaMohalla] = useState("");
  const [landmark, setLandmark] = useState("");
  const [postalCode, setPostalCode] = useState("");
  const [sameAsCurrent, setSameAsCurrent] = useState(false);
  const [stateID, setStateID] = useState(0);
  const [districtID, setDistrictID] = useState(0);
  const [subdistrictID, setSubdistrictID] = useState(0);
  const [localityID, setLocalityID] = useState(0);
  const [stateSearch, setStateSearch] = useState("");
  const [districtSearch, setDistrictSearch] = useState("");
  const [subdistrictSearch, setSubdistrictSearch] = useState("");
  const [localitySearch, setLocalitySearch] = useState("");
  const [showLocationCreator, setShowLocationCreator] = useState(false);
  const [newLocationName, setNewLocationName] = useState("");
  const [newLocalityType, setNewLocalityType] = useState("village");
  const [countryCode, setCountryCode] = useState("IN");
  const [fieldError, setFieldError] = useState("");
  const [feedback, setFeedback] = useState<{ kind: "success" | "error"; message: string } | null>(null);

  const addressesQuery = useSchoolStudentAddressesQuery(student.id);
  const addresses = addressesQuery.data?.addresses ?? [];
  const currentAddress = addresses.find(address => address.address_type === "current");
  const selectedAddress = addresses.find(address => address.address_type === addressType);
  const statesQuery = useSchoolStudentLocationsQuery({ level: "state", q: stateSearch });
  const districtsQuery = useSchoolStudentLocationsQuery({ parentID: stateID || undefined, level: "district", q: districtSearch }, stateID > 0);
  const subdistrictsQuery = useSchoolStudentLocationsQuery({ parentID: districtID || undefined, level: "subdistrict", q: subdistrictSearch }, districtID > 0);
  const localitiesQuery = useSchoolStudentLocationsQuery({ parentID: subdistrictID || undefined, level: "locality", q: localitySearch }, subdistrictID > 0);
  const saveAddress = useSaveSchoolStudentAddressMutation();
  const createLocation = useCreateSchoolStudentLocationMutation();

  useEffect(() => {
    setLine1(textValue(selectedAddress?.address_line_1));
    setLine2(textValue(selectedAddress?.address_line_2));
    setAreaMohalla(textValue(selectedAddress?.area_mohalla));
    setLandmark(textValue(selectedAddress?.landmark));
    setPostalCode(textValue(selectedAddress?.postal_code));
    setSameAsCurrent(Boolean(selectedAddress?.same_as_current));
    setStateID(selectedAddress?.state_location_id ?? 0);
    setDistrictID(selectedAddress?.district_location_id ?? 0);
    setSubdistrictID(selectedAddress?.subdistrict_location_id ?? 0);
    setLocalityID(selectedAddress?.location_id ?? 0);
    setFieldError("");
  }, [addressType, addressesQuery.data?.addresses]);

  const states = statesQuery.data?.locations ?? [];
  const districts = districtsQuery.data?.locations ?? [];
  const subdistricts = subdistrictsQuery.data?.locations ?? [];
  const localities = localitiesQuery.data?.locations ?? [];
  const hasAnyLocation = stateID > 0 || districtID > 0 || subdistrictID > 0 || localityID > 0;
  const incompleteLocation = hasAnyLocation && !(stateID > 0 && districtID > 0 && subdistrictID > 0 && localityID > 0);
  const canAddLocation = localityID === 0;
  const nextLocationLevel = !stateID ? "state" : !districtID ? "district" : !subdistrictID ? "subdistrict" : "locality";
  const nextLocationLabel = nextLocationLevel === "state" ? "State" :
    nextLocationLevel === "district" ? "District" :
    nextLocationLevel === "subdistrict" ? "Block / Tehsil" : "Village / City";
  const nextParentID = !stateID ? undefined : !districtID ? stateID : !subdistrictID ? districtID : subdistrictID;
  const busy = saveAddress.isPending || createLocation.isPending;

  function resetChildren(level: "state" | "district" | "subdistrict") {
    if (level === "state") {
      setDistrictID(0);
      setSubdistrictID(0);
      setLocalityID(0);
      setDistrictSearch("");
      setSubdistrictSearch("");
      setLocalitySearch("");
    } else if (level === "district") {
      setSubdistrictID(0);
      setLocalityID(0);
      setSubdistrictSearch("");
      setLocalitySearch("");
    } else {
      setLocalityID(0);
      setLocalitySearch("");
    }
    setFieldError("");
  }

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setFieldError("");
    if (addressType === "permanent" && sameAsCurrent && !currentAddress) {
      setFieldError("Save the current address before using it as the permanent address.");
      return;
    }
    if (!(addressType === "permanent" && sameAsCurrent) && incompleteLocation) {
      setFieldError("Complete State → District → Block / Tehsil → Village / City, or clear the location selection.");
      return;
    }
    try {
      await saveAddress.mutateAsync({
        studentID: student.id,
        addressType,
        input: sameAsCurrent && addressType === "permanent"
          ? { same_as_current: true }
          : {
              address_line_1: line1.trim() || null,
              address_line_2: line2.trim() || null,
              area_mohalla: areaMohalla.trim() || null,
              landmark: landmark.trim() || null,
              postal_code: postalCode.trim() || null,
              location_id: localityID || null,
              same_as_current: false,
            },
      });
      setFeedback({ kind: "success", message: addressType === "current" ? "Current address saved." : "Permanent address saved." });
    } catch (error) {
      setFeedback({ kind: "error", message: error instanceof Error ? error.message : "Address could not be saved." });
    }
  }

  async function addLocation() {
    setFieldError("");
    if (!newLocationName.trim()) {
      setFieldError("Enter the new location name.");
      return;
    }
    try {
      const result = await createLocation.mutateAsync({
        parent_id: nextParentID ?? null,
        country_code: nextLocationLevel === "state" ? countryCode.trim().toUpperCase() : undefined,
        level: nextLocationLevel,
        name: newLocationName.trim(),
        locality_type: nextLocationLevel === "locality" ? newLocalityType : null,
      }) as { location?: { id: number } };
      const id = result.location?.id;
      if (!id) throw new Error("The new location was not returned by the server.");
      if (nextLocationLevel === "state") {
        setStateID(id);
        resetChildren("state");
      } else if (nextLocationLevel === "district") {
        setDistrictID(id);
        resetChildren("district");
      } else if (nextLocationLevel === "subdistrict") {
        setSubdistrictID(id);
        resetChildren("subdistrict");
      } else {
        setLocalityID(id);
      }
      setNewLocationName("");
      setShowLocationCreator(false);
      setFeedback({ kind: "success", message: nextLocationLabel + " added to this school's location list." });
    } catch (error) {
      setFeedback({ kind: "error", message: error instanceof Error ? error.message : "Location could not be added." });
    }
  }

  return (
    <Modal
      open
      title="Address & Communication"
      description={student.full_name + " · Current and permanent addresses are saved independently."}
      onClose={onClose}
      closeDisabled={busy}
    >
      {addressesQuery.isError && <p className="student-feedback error" role="alert">{addressesQuery.error.message}</p>}
      {feedback && <p className={feedback.kind === "error" ? "student-feedback error" : "student-feedback success"} role={feedback.kind === "error" ? "alert" : "status"}>{feedback.message}</p>}
      <div className="student-address-tabs" role="group" aria-label="Address type">
        <button type="button" className={addressType === "current" ? "active" : ""} aria-pressed={addressType === "current"} onClick={() => { setAddressType("current"); setFeedback(null); }}>Current address</button>
        <button type="button" className={addressType === "permanent" ? "active" : ""} aria-pressed={addressType === "permanent"} onClick={() => { setAddressType("permanent"); setFeedback(null); }}>Permanent address</button>
      </div>
      <form className="student-address-form" onSubmit={submit} noValidate>
        {addressType === "permanent" && (
          <label className="student-address-same">
            <input type="checkbox" checked={sameAsCurrent} disabled={!currentAddress || busy} onChange={event => { setSameAsCurrent(event.target.checked); setFieldError(""); }} />
            Permanent address is same as current address
          </label>
        )}
        {sameAsCurrent && addressType === "permanent" ? (
          <p className="student-address-note">The permanent address will use the current address. No duplicate address details are stored.</p>
        ) : (
          <>
            <div className="student-address-grid">
              <label>House / street<input value={line1} maxLength={200} onChange={event => setLine1(event.target.value)} placeholder="House number, street or building"/></label>
              <label>Address line 2<input value={line2} maxLength={200} onChange={event => setLine2(event.target.value)} placeholder="Apartment, floor or additional details"/></label>
              <label>Area / mohalla<input value={areaMohalla} maxLength={120} onChange={event => setAreaMohalla(event.target.value)} placeholder="Local area or mohalla"/></label>
              <label>Landmark<input value={landmark} maxLength={120} onChange={event => setLandmark(event.target.value)} placeholder="Nearby landmark"/></label>
              <label>PIN / postal code<input value={postalCode} maxLength={20} onChange={event => setPostalCode(event.target.value)} placeholder="PIN or postal code"/></label>
            </div>
            <section className="student-location-section" aria-label="Location hierarchy">
              <h3>Location</h3>
              <p>Choose the full location path. If the exact village or city is unknown, leave all four selectors blank and use the address text fields.</p>
              <div className="student-location-grid">
                <label>State<input value={stateSearch} onChange={event => setStateSearch(event.target.value)} placeholder="Search states"/><Select ariaLabel="State" value={String(stateID || "")} onChange={value => { setStateID(Number(value)); resetChildren("state"); }} placeholder="Select state" options={states.map(item => ({ value: String(item.id), label: item.name }))}/></label>
                <label>District<input value={districtSearch} onChange={event => setDistrictSearch(event.target.value)} placeholder="Search districts" disabled={!stateID}/><Select ariaLabel="District" value={String(districtID || "")} onChange={value => { setDistrictID(Number(value)); resetChildren("district"); }} placeholder="Select district" disabled={!stateID || districtsQuery.isLoading} options={districts.map(item => ({ value: String(item.id), label: item.name }))}/></label>
                <label>Block / Tehsil<input value={subdistrictSearch} onChange={event => setSubdistrictSearch(event.target.value)} placeholder="Search blocks / tehsils" disabled={!districtID}/><Select ariaLabel="Block or Tehsil" value={String(subdistrictID || "")} onChange={value => { setSubdistrictID(Number(value)); resetChildren("subdistrict"); }} placeholder="Select block / tehsil" disabled={!districtID || subdistrictsQuery.isLoading} options={subdistricts.map(item => ({ value: String(item.id), label: item.name }))}/></label>
                <label>Village / City<input value={localitySearch} onChange={event => setLocalitySearch(event.target.value)} placeholder="Search villages / cities" disabled={!subdistrictID}/><Select ariaLabel="Village or City" value={String(localityID || "")} onChange={value => { setLocalityID(Number(value)); setFieldError(""); }} placeholder="Select village / city" disabled={!subdistrictID || localitiesQuery.isLoading} options={localities.map(item => ({ value: String(item.id), label: item.name + (item.locality_type ? " · " + item.locality_type : "") }))}/></label>
              </div>
              {fieldError && <small className="app-field-error">{fieldError}</small>}
              {canAddLocation && <button type="button" className="student-link-button" onClick={() => { setShowLocationCreator(value => !value); setFieldError(""); }}>Add missing {nextLocationLabel.toLowerCase()}</button>}
              {hasAnyLocation && <button type="button" className="student-link-button" onClick={() => { setStateID(0); resetChildren("state"); }}>Clear location selection</button>}
              {(statesQuery.isError || districtsQuery.isError || subdistrictsQuery.isError || localitiesQuery.isError) && <small className="app-field-error" role="alert">Location options could not be loaded. Close and reopen this section to retry.</small>}
              {showLocationCreator && canAddLocation && (
                <div className="student-location-create">
                  <strong>Add {nextLocationLabel.toLowerCase()} for this school</strong>
                  <label>Location name<input value={newLocationName} maxLength={120} onChange={event => setNewLocationName(event.target.value)} placeholder={"Enter " + nextLocationLabel.toLowerCase()}/></label>
                  {nextLocationLevel === "state" && <label>Country code<input value={countryCode} maxLength={2} onChange={event => setCountryCode(event.target.value.toUpperCase())} placeholder="IN"/></label>}
                  {nextLocationLevel === "locality" && <label>Locality type<Select ariaLabel="Locality type" value={newLocalityType} onChange={setNewLocalityType} options={[{value:"village",label:"Village"},{value:"city",label:"City"},{value:"town",label:"Town"},{value:"other",label:"Other"}]}/></label>}
                  <button type="button" className="secondary" disabled={createLocation.isPending} onClick={() => void addLocation()}>{createLocation.isPending ? "Adding…" : "Add location"}</button>
                </div>
              )}
            </section>
          </>
        )}
        <div className="app-modal-actions">
          <button type="button" className="app-modal-cancel" onClick={onClose} disabled={busy}>Close</button>
          <button className="primary" disabled={busy || addressesQuery.isLoading || addressesQuery.isError}>{saveAddress.isPending ? "Saving…" : "Save address"}</button>
        </div>
      </form>
    </Modal>
  );
}
