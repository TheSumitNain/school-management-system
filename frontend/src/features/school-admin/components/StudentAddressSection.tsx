import { useEffect, useState, type FormEvent } from "react";
import { Modal } from "../../../shared/components/Modal";
import { Select } from "../../../shared/components/Select";
import type { StudentLocation, StudentProfile } from "../api";
import {
  useCreateSchoolStudentLocationMutation,
  useDeactivateSchoolStudentLocationMutation,
  useSaveSchoolStudentAddressMutation,
  useSaveSchoolStudentContactDetailsMutation,
  useSchoolStudentAddressesQuery,
  useSchoolStudentContactDetailsQuery,
  useSchoolStudentLocationsQuery,
} from "../queries";

type AddressType = "current" | "permanent";

type Props = {
  student: StudentProfile;
  canUpdate: boolean;
  onClose: () => void;
};

function textValue(value?: string | null) {
  return value ?? "";
}

export function StudentAddressSection({ student, canUpdate, onClose }: Props) {
  const [addressType, setAddressType] = useState<AddressType>("current");
  const [showContactDetails, setShowContactDetails] = useState(false);
  const [primaryPhone, setPrimaryPhone] = useState("");
  const [alternatePhone, setAlternatePhone] = useState("");
  const [emailAddress, setEmailAddress] = useState("");
  const [contactErrors, setContactErrors] = useState<{ primaryPhone?: string; alternatePhone?: string; emailAddress?: string }>({});
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
  const [pendingDeactivate, setPendingDeactivate] = useState<StudentLocation | null>(null);
  const [newLocationName, setNewLocationName] = useState("");
  const [newLocalityType, setNewLocalityType] = useState("village");
  const [countryCode, setCountryCode] = useState("IN");
  const [fieldError, setFieldError] = useState("");
  const [feedback, setFeedback] = useState<{ kind: "success" | "error"; message: string } | null>(null);

  const addressesQuery = useSchoolStudentAddressesQuery(student.id, !showContactDetails);
  const contactQuery = useSchoolStudentContactDetailsQuery(student.id, showContactDetails);
  const addresses = addressesQuery.data?.addresses ?? [];
  const currentAddress = addresses.find(address => address.address_type === "current");
  const selectedAddress = addresses.find(address => address.address_type === addressType);
  const statesQuery = useSchoolStudentLocationsQuery({ level: "state", q: stateSearch });
  const districtsQuery = useSchoolStudentLocationsQuery({ parentID: stateID || undefined, level: "district", q: districtSearch }, stateID > 0);
  const subdistrictsQuery = useSchoolStudentLocationsQuery({ parentID: districtID || undefined, level: "subdistrict", q: subdistrictSearch }, districtID > 0);
  const localitiesQuery = useSchoolStudentLocationsQuery({ parentID: subdistrictID || undefined, level: "locality", q: localitySearch }, subdistrictID > 0);
  const saveAddress = useSaveSchoolStudentAddressMutation();
  const saveContact = useSaveSchoolStudentContactDetailsMutation();
  const createLocation = useCreateSchoolStudentLocationMutation();
  const deactivateLocation = useDeactivateSchoolStudentLocationMutation();

  useEffect(() => {
    if (showContactDetails) return;
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
  }, [showContactDetails, addressType, addressesQuery.data?.addresses]);

  useEffect(() => {
    if (!showContactDetails) return;
    const details = contactQuery.data?.contact_details;
    setPrimaryPhone(textValue(details?.primary_phone));
    setAlternatePhone(textValue(details?.alternate_phone));
    setEmailAddress(textValue(details?.email_address));
  }, [showContactDetails, contactQuery.data?.contact_details]);

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
  const selectedLocation = localityID ? localities.find(item => item.id === localityID) :
    subdistrictID ? subdistricts.find(item => item.id === subdistrictID) :
    districtID ? districts.find(item => item.id === districtID) :
    stateID ? states.find(item => item.id === stateID) : undefined;
  const busy = saveAddress.isPending || saveContact.isPending || createLocation.isPending || deactivateLocation.isPending;

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
    if (!canUpdate) return;
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

  async function submitContact(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!canUpdate || contactQuery.isLoading) return;
    const errors: { primaryPhone?: string; alternatePhone?: string; emailAddress?: string } = {};
    if (primaryPhone.trim().length > 30) errors.primaryPhone = "Phone number must be 30 characters or fewer.";
    if (alternatePhone.trim().length > 30) errors.alternatePhone = "Phone number must be 30 characters or fewer.";
    const emailInput = event.currentTarget.elements.namedItem("student_email_address") as HTMLInputElement | null;
    if (emailAddress.trim() && emailInput && !emailInput.checkValidity()) {
      errors.emailAddress = "Enter a valid email address.";
    }
    setContactErrors(errors);
    if (Object.keys(errors).length > 0) return;
    try {
      await saveContact.mutateAsync({
        studentID: student.id,
        input: {
          primary_phone: primaryPhone.trim() || null,
          alternate_phone: alternatePhone.trim() || null,
          email_address: emailAddress.trim() || null,
        },
      });
      setFeedback({ kind: "success", message: "Student contact details saved." });
    } catch (error) {
      setFeedback({ kind: "error", message: error instanceof Error ? error.message : "Contact details could not be saved." });
    }
  }

  async function confirmDeactivateLocation() {
    if (!canUpdate || !pendingDeactivate) return;
    try {
      await deactivateLocation.mutateAsync(pendingDeactivate.id);
      const deactivated = pendingDeactivate;
      setPendingDeactivate(null);
      if (deactivated.level === "state") {
        setStateID(0);
        resetChildren("state");
      } else if (deactivated.level === "district") {
        setDistrictID(0);
        resetChildren("district");
      } else if (deactivated.level === "subdistrict") {
        setSubdistrictID(0);
        resetChildren("subdistrict");
      } else if (deactivated.level === "locality" && selectedAddress?.location_id !== deactivated.id) {
        setLocalityID(0);
      }
      setFeedback({ kind: "success", message: deactivated.name + " was deactivated. Existing address records are retained." });
    } catch (error) {
      setFeedback({ kind: "error", message: error instanceof Error ? error.message : "Location could not be deactivated." });
    }
  }

  async function addLocation() {
    if (!canUpdate) return;
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
      {!showContactDetails && addressesQuery.isError && <p className="student-feedback error" role="alert">{addressesQuery.error.message}</p>}
      {feedback && <p className={feedback.kind === "error" ? "student-feedback error" : "student-feedback success"} role={feedback.kind === "error" ? "alert" : "status"}>{feedback.message}</p>}
      <div className="student-address-tabs" role="group" aria-label="Student profile section">
        <button type="button" className={!showContactDetails && addressType === "current" ? "active" : ""} aria-pressed={!showContactDetails && addressType === "current"} onClick={() => { setShowContactDetails(false); setAddressType("current"); setFeedback(null); }}>Current address</button>
        <button type="button" className={!showContactDetails && addressType === "permanent" ? "active" : ""} aria-pressed={!showContactDetails && addressType === "permanent"} onClick={() => { setShowContactDetails(false); setAddressType("permanent"); setFeedback(null); }}>Permanent address</button>
        <button type="button" className={showContactDetails ? "active" : ""} aria-pressed={showContactDetails} onClick={() => { setShowContactDetails(true); setFeedback(null); setContactErrors({}); }}>Contact details</button>
      </div>
      {showContactDetails ? (
        <form className="student-address-form" onSubmit={submitContact} noValidate>
          {contactQuery.isError && <p className="student-feedback error" role="alert">{contactQuery.error.message}</p>}
          <div className="student-address-grid">
            <label>Primary phone<input disabled={!canUpdate || contactQuery.isLoading || busy} value={primaryPhone} maxLength={30} onChange={event => { setPrimaryPhone(event.target.value); setContactErrors(current => ({ ...current, primaryPhone: undefined })); }} placeholder="Primary contact number"/>{contactErrors.primaryPhone && <small className="app-field-error">{contactErrors.primaryPhone}</small>}</label>
            <label>Alternate phone<input disabled={!canUpdate || contactQuery.isLoading || busy} value={alternatePhone} maxLength={30} onChange={event => { setAlternatePhone(event.target.value); setContactErrors(current => ({ ...current, alternatePhone: undefined })); }} placeholder="Alternate contact number"/>{contactErrors.alternatePhone && <small className="app-field-error">{contactErrors.alternatePhone}</small>}</label>
            <label>Email address<input name="student_email_address" disabled={!canUpdate || contactQuery.isLoading || busy} type="email" value={emailAddress} maxLength={254} aria-invalid={Boolean(contactErrors.emailAddress)} aria-describedby={contactErrors.emailAddress ? "student-contact-email-error" : undefined} onChange={event => { setEmailAddress(event.target.value); setContactErrors(current => ({ ...current, emailAddress: undefined })); }} placeholder="student@example.com"/>{contactErrors.emailAddress && <small id="student-contact-email-error" className="app-field-error">{contactErrors.emailAddress}</small>}</label>
          </div>
          {!canUpdate && <p className="student-address-note">View-only access. Student update permission is required to edit contact details.</p>}
          <div className="app-modal-actions">
            <button type="button" className="app-modal-cancel" onClick={onClose} disabled={busy}>Close</button>
            {canUpdate && <button className="primary" disabled={busy || contactQuery.isLoading || contactQuery.isError}>{saveContact.isPending ? "Saving…" : "Save contact details"}</button>}
          </div>
        </form>
      ) : (
      <form className="student-address-form" onSubmit={submit} noValidate>
        {addressType === "permanent" && (
          <label className="student-address-same">
            <input type="checkbox" checked={sameAsCurrent} disabled={!canUpdate || !currentAddress || busy} onChange={event => { setSameAsCurrent(event.target.checked); setFieldError(""); }} />
            Permanent address is same as current address
          </label>
        )}
        {sameAsCurrent && addressType === "permanent" ? (
          <p className="student-address-note">The permanent address will use the current address. No duplicate address details are stored.</p>
        ) : (
          <>
            <div className="student-address-grid">
              <label>House / street<input disabled={!canUpdate} value={line1} maxLength={200} onChange={event => setLine1(event.target.value)} placeholder="House number, street or building"/></label>
              <label>Address line 2<input disabled={!canUpdate} value={line2} maxLength={200} onChange={event => setLine2(event.target.value)} placeholder="Apartment, floor or additional details"/></label>
              <label>Area / mohalla<input disabled={!canUpdate} value={areaMohalla} maxLength={120} onChange={event => setAreaMohalla(event.target.value)} placeholder="Local area or mohalla"/></label>
              <label>Landmark<input disabled={!canUpdate} value={landmark} maxLength={120} onChange={event => setLandmark(event.target.value)} placeholder="Nearby landmark"/></label>
              <label>PIN / postal code<input disabled={!canUpdate} value={postalCode} maxLength={20} onChange={event => setPostalCode(event.target.value)} placeholder="PIN or postal code"/></label>
            </div>
            <section className="student-location-section" aria-label="Location hierarchy">
              <h3>Location</h3>
              <p>Choose the full location path. If the exact village or city is unknown, leave all four selectors blank and use the address text fields.</p>
              <div className="student-location-grid">
                <label>State<input disabled={!canUpdate} value={stateSearch} onChange={event => setStateSearch(event.target.value)} placeholder="Search states"/><Select ariaLabel="State" value={String(stateID || "")} disabled={!canUpdate} onChange={value => { setStateID(Number(value)); resetChildren("state"); }} placeholder="Select state" options={states.map(item => ({ value: String(item.id), label: item.name }))}/></label>
                <label>District<input disabled={!canUpdate || !stateID} value={districtSearch} onChange={event => setDistrictSearch(event.target.value)} placeholder="Search districts"/><Select ariaLabel="District" value={String(districtID || "")} disabled={!canUpdate || !stateID || districtsQuery.isLoading} onChange={value => { setDistrictID(Number(value)); resetChildren("district"); }} placeholder="Select district" options={districts.map(item => ({ value: String(item.id), label: item.name }))}/></label>
                <label>Block / Tehsil<input disabled={!canUpdate || !districtID} value={subdistrictSearch} onChange={event => setSubdistrictSearch(event.target.value)} placeholder="Search blocks / tehsils"/><Select ariaLabel="Block or Tehsil" value={String(subdistrictID || "")} disabled={!canUpdate || !districtID || subdistrictsQuery.isLoading} onChange={value => { setSubdistrictID(Number(value)); resetChildren("subdistrict"); }} placeholder="Select block / tehsil" options={subdistricts.map(item => ({ value: String(item.id), label: item.name }))}/></label>
                <label>Village / City<input disabled={!canUpdate || !subdistrictID} value={localitySearch} onChange={event => setLocalitySearch(event.target.value)} placeholder="Search villages / cities"/><Select ariaLabel="Village or City" value={String(localityID || "")} disabled={!canUpdate || !subdistrictID || localitiesQuery.isLoading} onChange={value => { setLocalityID(Number(value)); setFieldError(""); }} placeholder="Select village / city" options={localities.map(item => ({ value: String(item.id), label: item.name + (item.locality_type ? " · " + item.locality_type : "") }))}/></label>
              </div>
              {selectedAddress?.location_name && <p className="student-address-note">Saved locality: {selectedAddress.location_name}{localityID > 0 && !localities.some(item => item.id === localityID) ? " (not in the active location list)" : ""}.</p>}
              {fieldError && <small className="app-field-error">{fieldError}</small>}
              {canUpdate && canAddLocation && <button type="button" className="student-link-button" onClick={() => { setShowLocationCreator(value => !value); setFieldError(""); }}>Add missing {nextLocationLabel.toLowerCase()}</button>}
              {canUpdate && hasAnyLocation && <button type="button" className="student-link-button" onClick={() => { setStateID(0); resetChildren("state"); }}>Clear location selection</button>}
              {canUpdate && selectedLocation?.school_id && <button type="button" className="student-link-button" onClick={() => setPendingDeactivate(selectedLocation)}>Deactivate selected school location</button>}
              {pendingDeactivate && <div className="student-location-confirm" role="alertdialog" aria-label="Confirm location deactivation">
                <p>Deactivate <strong>{pendingDeactivate.name}</strong>? It will disappear from new selections. Existing student addresses will keep their saved location.</p>
                <div><button type="button" className="app-modal-cancel" onClick={() => setPendingDeactivate(null)} disabled={busy}>Cancel</button><button type="button" className="secondary" onClick={() => void confirmDeactivateLocation()} disabled={busy}>{deactivateLocation.isPending ? "Deactivating…" : "Confirm deactivation"}</button></div>
              </div>}
              {(statesQuery.isError || districtsQuery.isError || subdistrictsQuery.isError || localitiesQuery.isError) && <small className="app-field-error" role="alert">Location options could not be loaded. Close and reopen this section to retry.</small>}
              {showLocationCreator && canUpdate && canAddLocation && (
                <div className="student-location-create">
                  <strong>Add {nextLocationLabel.toLowerCase()} for this school</strong>
                  <label>Location name<input disabled={!canUpdate} value={newLocationName} maxLength={120} onChange={event => setNewLocationName(event.target.value)} placeholder={"Enter " + nextLocationLabel.toLowerCase()}/></label>
                  {nextLocationLevel === "state" && <label>Country code<input disabled={!canUpdate} value={countryCode} maxLength={2} onChange={event => setCountryCode(event.target.value.toUpperCase())} placeholder="IN"/></label>}
                  {nextLocationLevel === "locality" && <label>Locality type<Select ariaLabel="Locality type" value={newLocalityType} disabled={!canUpdate} onChange={setNewLocalityType} options={[{value:"village",label:"Village"},{value:"city",label:"City"},{value:"town",label:"Town"},{value:"other",label:"Other"}]}/></label>}
                  <button type="button" className="secondary" disabled={createLocation.isPending} onClick={() => void addLocation()}>{createLocation.isPending ? "Adding…" : "Add location"}</button>
                </div>
              )}
            </section>
          </>
        )}
        {!canUpdate && <p className="student-address-note">View-only access. Student update permission is required to edit addresses or add locations.</p>}
        <div className="app-modal-actions">
          <button type="button" className="app-modal-cancel" onClick={onClose} disabled={busy}>Close</button>
          {canUpdate && <button className="primary" disabled={busy || addressesQuery.isLoading || addressesQuery.isError}>{saveAddress.isPending ? "Saving…" : "Save address"}</button>}
        </div>
      </form>
      )}
    </Modal>
  );
}
