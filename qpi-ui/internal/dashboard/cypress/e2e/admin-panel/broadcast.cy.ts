describe("Admin Panel — Broadcast Announcement", () => {
  beforeEach(() => {
    cy.clearCookies();
    cy.clearLocalStorage();
    cy.visit("/");

    cy.contains("button", "Administrator").click();
    cy.get('input[type="text"]').type("admin@example.com");
    cy.get('input[type="password"]').type("supersecretpassword1234");
    cy.get('button[type="submit"]').click();
    cy.contains("h1", "QPI Interface").should("be.visible");

    cy.contains("button", "Admin Panel").click();
    cy.contains("h1", "Admin Panel").should("be.visible");

    cy.contains("button", "Broadcast Announcement").click();
    cy.contains("button", "Broadcast Announcement")
      .should("have.class", "border-b-2");
  });

  it("composes and broadcasts an announcement, then sees it in the bell dropdown", () => {
    const title = "Scheduled Maintenance";
    const description = "The QPU will be offline for calibration tomorrow.";

    // Fill the broadcast form
    cy.get('input[placeholder="QPU Maintenance Schedule"]')
      .type(title);
    cy.get('textarea[placeholder="Rigetti Aspen-9 will be offline for calibration tomorrow..."]')
      .type(description);

    // Submit the form
    cy.contains("button", "Broadcast Announcement").click();

    // Verify success message
    cy.contains("Announcement broadcasted successfully!").should("be.visible");

    // Open the notification bell dropdown
    cy.get('button svg.lucide-bell')
      .parent()
      .click();

    // The exact title should appear in the dropdown
    cy.contains("p", title).should("be.visible");
    cy.contains("p", description).should("be.visible");
  });
});
