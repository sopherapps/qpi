describe("Overview & Header — Notifications Panel", () => {
  beforeEach(() => {
    cy.clearCookies();
    cy.clearLocalStorage();
    cy.visit("/");

    cy.contains("button", "Administrator").click();
    cy.get('input[type="text"]').clear().type("admin@example.com");
    cy.get('input[type="password"]').clear().type("supersecretpassword1234");
    cy.get('button[type="submit"]').click();
    cy.contains("h1", "QPI Interface").should("be.visible");

    // Broadcast an announcement so there's something to dismiss
    cy.contains("button", "Admin Panel").click();
    cy.contains("h1", "Admin Panel").should("be.visible");
    cy.contains("button", "Broadcast Announcement").click();

    const uniqueTitle = `Test Announcement`;
    cy.wrap(uniqueTitle).as('annTitle');

    cy.get('input[placeholder="QPU Maintenance Schedule"]')
      .clear().type(uniqueTitle);
    cy.get('textarea[placeholder="Rigetti Aspen-9 will be offline for calibration tomorrow..."]')
      .clear().type(`This is a test announcement for the notifications panel.`);
    
    // Fill datetime
    const pad = (n: number) => n.toString().padStart(2, "0");
    const start = new Date(Date.now() - 5 * 60000);
    const end = new Date(Date.now() + 20 * 60000);
    const startStr = `${start.getFullYear()}-${pad(start.getMonth() + 1)}-${pad(start.getDate())}T${pad(start.getHours())}:${pad(start.getMinutes())}`;
    const endStr = `${end.getFullYear()}-${pad(end.getMonth() + 1)}-${pad(end.getDate())}T${pad(end.getHours())}:${pad(end.getMinutes())}`;

    cy.get('input[type="datetime-local"]').first().type(startStr);
    cy.get('input[type="datetime-local"]').last().type(endStr);

    cy.get("form").contains("button", "Broadcast Announcement").click();

    cy.contains("Announcement broadcasted successfully!").should("be.visible");

    // Go back to Overview
    cy.contains("button", "Overview").click();
    cy.contains("h1", "Overview").should("be.visible");
  });

  it("dismisses an individual notification and it disappears from the panel", function() {
    const title = this.annTitle;
    // The notification should be visible in the System Announcements panel
    cy.contains("h3", "System Announcements")
      .parent()
      .within(() => {
        cy.contains(title).scrollIntoView().should("be.visible");
      });

    // Hover over the notification to reveal the dismiss button, then click it
    cy.contains("h3", "System Announcements")
      .parent()
      .within(() => {
        cy.contains(title)
          .parent()
          .parent()
          .parent()
          .trigger("mouseover")
          .within(() => {
            cy.get('button svg.lucide-x').parent().click({ force: true });
          });
      });

    // The notification should disappear
    cy.contains("h3", "System Announcements")
      .parent()
      .within(() => {
        cy.contains(title).should("not.exist");
      });
  });

  it('dismisses all notifications when "Clear All" is clicked from the header bell', () => {
    // Open the header bell dropdown
    cy.get('button svg.lucide-bell')
      .parent()
      .click();

    // Click "Clear All"
    cy.contains("button", "Clear All").click();

    // The System Announcements panel should show the empty state
    cy.contains("h3", "System Announcements")
      .parent()
      .within(() => {
        cy.contains("No active announcements").scrollIntoView().should("be.visible");
      });
  });
});
