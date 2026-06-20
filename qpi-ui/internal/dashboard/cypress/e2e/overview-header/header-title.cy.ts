describe("Overview & Header — Page Title", () => {
  beforeEach(() => {
    cy.clearCookies();
    cy.clearLocalStorage();
    cy.visit("/");

    cy.get('input[type="text"]').type("user@example.com");
    cy.get('input[type="password"]').type("userpassword1234");
    cy.get('button[type="submit"]').click();
    cy.contains("h1", "QPI Interface").should("be.visible");
  });

  it('shows "Overview" as the header title on the default tab', () => {
    cy.get("header h2").should("contain", "Overview");
  });

  it('shows "QPU Registry" when on the QPUs tab', () => {
    cy.contains("button", "QPUs").click();
    cy.get("header h2").should("contain", "QPU Registry");
  });

  it('shows "Jobs Console" when on the Jobs tab', () => {
    cy.contains("button", "Jobs").click();
    cy.get("header h2").should("contain", "Jobs Console");
  });

  it('shows "Bookings Overview" when on the Bookings tab', () => {
    cy.contains("button", "Bookings").click();
    cy.get("header h2").should("contain", "Bookings Overview");
  });

  it('shows "Settings Overview" when on the Settings tab', () => {
    cy.contains("button", "Settings").click();
    cy.get("header h2").should("contain", "Settings Overview");
  });
});
